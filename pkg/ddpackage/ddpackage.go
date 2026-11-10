// Package ddpackage provides the Package structures and functions
package ddpackage

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	// "io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/sirupsen/logrus"

	dderrors "github.com/ryex/dungeondraft-gopackager/internal/errors"
)

type PackOptions struct {
	Overwrite bool
	ValidExts []string
}

type UnpackOptions struct {
	RipTextures bool
	Overwrite   bool
	Thumbnails  bool
}

type PackageMode int

const (
	PackageModeUnloaded PackageMode = iota
	PackageModePacked
	PackageModeUnpacked
)

type Package struct {
	log           logrus.FieldLogger
	mode          PackageMode
	name          string
	id            string
	unpackOptions *UnpackOptions
	packOptions   *PackOptions
	unpackedPath  string
	packedPath    string
	alignment     int

	// flLock guards the resourceMap and the fileList, and the pack info
	// (id, name, info), because the info is reloaded from pack.json while
	// the file list is scanned (see LoadUnpackedPackJSON).
	flLock      sync.RWMutex
	fileList    *structures.FileInfoList
	resourceMap map[string]*structures.FileInfo

	// md5Mu guards md5Inflight, the callbacks queued behind a running md5
	// computation. Lock order: flLock may be held while acquiring md5Mu,
	// never the reverse; md5Mu is never held while doing I/O or invoking
	// callbacks.
	md5Mu       sync.Mutex
	md5Inflight map[string][]func(string, error)
	// pkgFileMu serializes the seek+read of the shared packed file, which
	// moves its position, between the concurrent readers of it (the
	// preview, the extraction and the md5 computation). It is taken inside
	// readPackedFileFromPackage and is never held while another lock of
	// the package is acquired.
	pkgFileMu sync.Mutex

	info *structures.PackageInfo

	walls    map[string]structures.PackageWall
	tilesets map[string]structures.PackageTileset

	tags *structures.PackageTags

	pkgFile *os.File
}

func (p *Package) Close() {
	if p.pkgFile != nil {
		if p.pkgFile.Close() != nil {
		}
		p.pkgFile = nil
	}
}

func (p *Package) ID() string {
	p.flLock.RLock()
	defer p.flLock.RUnlock()
	return p.id
}

func (p *Package) SetID(id string) {
	p.flLock.Lock()
	defer p.flLock.Unlock()
	p.id = id
	p.info.ID = id
}

func (p *Package) Name() string {
	p.flLock.RLock()
	defer p.flLock.RUnlock()
	return p.name
}

func (p *Package) SetName(name string) {
	p.flLock.Lock()
	defer p.flLock.Unlock()
	p.name = name
	p.info.Name = name
}

func (p *Package) SetAuthor(author string) {
	p.flLock.Lock()
	defer p.flLock.Unlock()
	p.info.Author = author
}

func (p *Package) SetVersion(version string) {
	p.flLock.Lock()
	defer p.flLock.Unlock()
	p.info.Version = version
}

func (p *Package) SetKeywords(keywords []string) {
	p.flLock.Lock()
	defer p.flLock.Unlock()
	p.info.Keywords = keywords
	p.info.KeywordsRaw = strings.Join(keywords, ",")
}

func (p *Package) SetAllow3rdParty(allow *bool) {
	val := new(bool)
	if allow != nil {
		*val = *allow
	}
	p.flLock.Lock()
	defer p.flLock.Unlock()
	p.info.Allow3rdParty = val
}

func (p *Package) SetColorOverrides(overrides *structures.CustomColorOverrides) {
	p.flLock.Lock()
	defer p.flLock.Unlock()
	p.info.ColorOverrides = overrides
}

func (p *Package) UnpackedPath() string {
	return p.unpackedPath
}

func (p *Package) PackedPath() string {
	return p.packedPath
}

func (p *Package) FileList() *structures.FileInfoList {
	p.flLock.RLock()
	defer p.flLock.RUnlock()
	res := p.fileList.Copy()
	return res
}

// Info returns the info of the pack that was read from its pack.json.
//
// The info is replaced when it is read again, so the returned pointer stays
// usable while the package is updated.
func (p *Package) Info() *structures.PackageInfo {
	p.flLock.RLock()
	defer p.flLock.RUnlock()
	return p.info
}

func (p *Package) Walls() map[string]structures.PackageWall {
	return p.walls
}

func (p *Package) Tags() *structures.PackageTags {
	return p.tags
}

func (p *Package) Tilesets() map[string]structures.PackageTileset {
	return p.tilesets
}

func NewPackage(log logrus.FieldLogger) *Package {
	return &Package{
		log:         log,
		mode:        PackageModeUnloaded,
		alignment:   0,
		walls:       make(map[string]structures.PackageWall),
		tilesets:    make(map[string]structures.PackageTileset),
		resourceMap: make(map[string]*structures.FileInfo),
		tags:        structures.NewPackageTags(),
		fileList:    structures.NewFileInfoList(),
	}
}

// SetAlignment sets packed file alignment
func (p *Package) SetAlignment(alignment int) error {
	if alignment < 0 {
		return errors.New("alignment must be greater than 0")
	}
	p.alignment = alignment
	return nil
}

func (p *Package) SetUnpackOptions(options UnpackOptions) {
	p.unpackOptions = &options
}

func (p *Package) SetPackOptions(options PackOptions) {
	if len(options.ValidExts) == 0 {
		options.ValidExts = DefaultValidExt()
	}
	p.packOptions = &options
}

// DefaultValidExt returns a slice of valid file extensions for inclusion in a .dungeondraft_pack
func DefaultValidExt() []string {
	return []string{
		".png", ".webp", ".jpg", ".jpeg",
		".gif", ".tif", ".tiff", ".bmp",
		".dungeondraft_wall", ".dungeondraft_tileset",
		".dungeondraft_tags", ".json",
	}
}

func (p *Package) LoadFromPackedPath(
	path string,
	progressCallback func(p float64, curRes string, max int64),
) error {
	packFilePath, pathErr := filepath.Abs(path)
	if pathErr != nil {
		p.log.WithField("path", packFilePath).WithError(pathErr).Error("could not get absolute path for package file")
		return dderrors.CausedBy(errors.New("could not get absolute path for package file"), pathErr)
	}
	file, fileErr := os.Open(packFilePath)
	if fileErr != nil {
		p.log.WithField("path", packFilePath).WithError(fileErr).Error("could not open package file for reading")
		return dderrors.CausedBy(errors.New("could not open package file for reading"), fileErr)
	}

	p.SetUnpackOptions(UnpackOptions{})

	err := p.loadPackedFileList(file, progressCallback)
	if err != nil {
		p.log.WithError(err).Error("failed to read file list")
		file.Close()
		return dderrors.CausedBy(errors.New("failed to read file list"), err)
	}
	err = p.loadPackedPackJSON(file)
	if err != nil {
		p.log.WithError(err).Error("failed to read pack json")
		file.Close()
		return dderrors.CausedBy(errors.New("failed to read pack json"), err)
	}
	p.packedPath = packFilePath
	p.pkgFile = file
	p.mode = PackageModePacked

	return nil
}

func (p *Package) LoadUnpackedFromFolder(dirPath string) error {
	dirPath, pathErr := filepath.Abs(dirPath)
	if pathErr != nil {
		p.log.WithField("path", dirPath).
			WithError(pathErr).
			Error("could not get absolute path for package folder")
		return dderrors.CausedBy(errors.New("could not get absolute path for package folder"), pathErr)
	}

	if dirExists := utils.DirExists(dirPath); !dirExists {
		err := fmt.Errorf("path %s does not exists or is not a directory", dirPath)
		p.log.WithError(err).
			WithField("path", dirPath).
			Error("can't package a non existent folder")
		return err
	}

	if err := p.LoadUnpackedPackJSON(dirPath); err != nil {
		return err
	}

	p.SetPackOptions(PackOptions{})

	p.unpackedPath = dirPath
	p.mode = PackageModeUnpacked

	return nil
}

func (p *Package) RemoveResource(res string) {
	p.flLock.Lock()
	defer p.flLock.Unlock()
	p.removeResource(res)
}

// DO NOT USE WITHOUT flLock
func (p *Package) removeResource(res string) {
	p.fileList.RemoveRes(res)
	delete(p.resourceMap, res)
}

func (p *Package) AddResource(fInfo *structures.FileInfo) {
	p.flLock.Lock()
	defer p.flLock.Unlock()
	p.addResource(fInfo)
}

// DO NOT USE WITHOUT flLock
func (p *Package) addResource(fInfo *structures.FileInfo) {
	if _, ok := p.resourceMap[fInfo.ResPath]; ok {
		return
	}
	p.fileList.Add(fInfo)
	p.resourceMap[fInfo.ResPath] = fInfo
	p.log.Debugf("added %s", fInfo.ResPath)
}

// replaceResource replaces the info of a resource with the given info.
//
// It is used when the file of a resource changed on disk, so that its size
// and hash are picked up again.
//
// DO NOT USE WITHOUT flLock
func (p *Package) replaceResource(old *structures.FileInfo, fInfo *structures.FileInfo) {
	if old != nil {
		if index := p.fileList.IndexOfRes(old.ResPath); index != -1 {
			p.fileList.Remove(index)
		}
	}
	if index := p.fileList.IndexOfRes(fInfo.ResPath); index != -1 {
		p.fileList.Remove(index)
	}
	p.fileList.Add(fInfo)
	p.resourceMap[fInfo.ResPath] = fInfo
	p.log.Debugf("replaced %s", fInfo.ResPath)
}

// resetData clears the resource index of the package.
//
// The tags and the resource metadata are not cleared here: they are cleared
// in place when they are read again (resetTags, resetResourceMetadata). The
// interface reads them without a lock, so they must not be written by the
// scan of the file list, which can run in the background.
func (p *Package) resetData() {
	p.flLock.Lock()
	defer p.flLock.Unlock()
	p.fileList = structures.NewFileInfoList()
	p.resourceMap = make(map[string]*structures.FileInfo)
}

// GetResourceInfo gets the *FileInfo for a resource identified by the passed 'res://' path
func (p *Package) GetResourceInfo(resPath string) (*structures.FileInfo, error) {
	p.flLock.RLock()
	defer p.flLock.RUnlock()

	info, found := p.resourceMap[resPath]
	if found && info != nil {
		return info, nil
	}
	info = p.fileList.Find(func(fi *structures.FileInfo) bool {
		return fi.ResPath == resPath
	})
	if info != nil {
		return info, nil
	}
	return nil, ErrResourceNotFound
}

func (p *Package) ReadPackJSON() error {
	switch p.mode {
	case PackageModePacked:
		return p.loadPackedPackJSON(p.pkgFile)
	case PackageModeUnpacked:
		return p.LoadUnpackedPackJSON(p.unpackedPath)
	}
	return ErrPackageNotLoaded
}

func (p *Package) LoadTags() error {
	switch p.mode {
	case PackageModePacked:
		return p.loadPackedTags(p.pkgFile)
	case PackageModeUnpacked:
		return p.loadUnpackedTags()
	}
	return ErrPackageNotLoaded
}

func (p *Package) LoadResourceMetadata() error {
	switch p.mode {
	case PackageModePacked:
		return p.loadPackedResourceMetadata(p.pkgFile)
	case PackageModeUnpacked:
		return p.loadUnpackedResourceMetadata()
	}
	return ErrPackageNotLoaded
}

// LoadResource Loads the resource identified by the passed 'res://' path
func (p *Package) LoadResource(resPath string) ([]byte, error) {
	if p.mode != PackageModePacked && p.mode != PackageModeUnpacked {
		return nil, ErrPackageNotLoaded
	}

	info, error := p.GetResourceInfo(resPath)
	if error != nil {
		return nil, error
	}

	if p.mode == PackageModePacked {
		return p.readPackedFileFromPackage(p.pkgFile, info)
	} else {
		return p.readUnpackedFileFromPackage(info)
	}
}

type NewFileInfoOptions struct {
	Path    string
	ResPath *string
	RelPath *string
	Size    int64
}

func (p *Package) NewFileInfo(options NewFileInfoOptions) (*structures.FileInfo, error) {
	if p.unpackedPath == "" {
		return nil, ErrUnsetUnpackedPath
	}

	if options.Path != "" && (options.ResPath == nil || options.RelPath == nil) {

		l := p.log.WithField("filePath", options.Path)
		relPath, err := filepath.Rel(p.unpackedPath, options.Path)
		if err != nil {
			l.Error("can not get path relative to package root")
			return nil, err
		}

		if options.ResPath == nil {
			options.ResPath = new(string)
			*options.ResPath = fmt.Sprintf("res://packs/%s/%s", p.id, relPath)

			if runtime.GOOS == "windows" { // windows path separators.....
				*options.ResPath = strings.ReplaceAll(*options.ResPath, "\\", "/")
			}
		}

		if options.RelPath == nil {
			options.RelPath = new(string)
			if runtime.GOOS == "windows" { // windows path separators.....
				relPath = strings.ReplaceAll(relPath, "\\", "/")
			}
			*options.RelPath = relPath
		}
	}

	info := &structures.FileInfo{
		Path:        options.Path,
		ResPath:     *options.ResPath,
		RelPath:     *options.RelPath,
		ResPathSize: int32(len([]byte(*options.ResPath))),
		Size:        options.Size,
	}

	if options.Path != "" {

		l := p.log.WithField("filePath", options.Path)

		if info.IsTexture() {

			thumbnailDir := filepath.Join(p.unpackedPath, "thumbnails")
			hash := md5.Sum([]byte(info.ResPath))
			thumbnailName := strings.ToLower(hex.EncodeToString(hash[:])) + ".png"
			thumbnailPath := filepath.Join(thumbnailDir, thumbnailName)
			info.ThumbnailPath = thumbnailPath
			info.ThumbnailResPath = fmt.Sprintf("res://packs/%s/thumbnails/%s", p.id, thumbnailName)

			if !ddimage.PathIsSupportedDDImage(options.Path) {
				img, format, err := ddimage.OpenImage(options.Path)
				if err != nil {
					l.WithError(err).Error("can not open path with image extension as image")
					// log but let info construction continue
				} else {
					l.WithField("imageFormat", format).Trace("read image")
					info.ImageFormat = format

					info.Image = img
					l.WithField("imageFormat", format).
						Info("format is not supported by dungeondraft, converting to png")
					buf := new(bytes.Buffer)
					err = ddimage.PngImageBytes(img, buf)
					if err != nil {
						l.WithError(err).Error("failed to encode png version of image")
						// log but let info construction continue
					} else {
						imgBytes := buf.Bytes()
						info.PngImage = make([]byte, len(imgBytes))
						copy(info.PngImage, imgBytes)

						info.Size = int64(len(info.PngImage))

						ext := filepath.Ext(options.Path)
						info.ResPath = info.ResPath[0:len(info.ResPath)-len(ext)] + ".png"
						info.RelPath = info.RelPath[0:len(info.RelPath)-len(ext)] + ".png"
					}
				}
			}

			isWall := info.IsWall()
			isTileset := info.IsTileset()

			if isWall || isTileset {
				fName := filepath.Base(info.RelPath)
				bName := fName[:len(fName)-len(filepath.Ext(fName))]

				if isWall {
					info.MetadataPath = fmt.Sprintf("res://packs/%s/data/walls/%s.dungeondraft_wall", p.id, bName)
				} else {
					info.MetadataPath = fmt.Sprintf("res://packs/%s/data/tilesets/%s.dungeondraft_tileset", p.id, bName)
				}
			}

		}
	}

	return info, nil
}

// GetOrUpdateResourceMd5 reports the md5 hash of fi through callback. When the
// hash is not known yet it is computed once per resource: further callers for
// the same resource queue behind the inflight computation (single-flight) and
// every callback receives the same result. The hash is published under flLock
// before the callbacks run, so a callback always observes a valid fi.Md5.
func (p *Package) GetOrUpdateResourceMd5(fi *structures.FileInfo, callback func(string, error)) {
	key := fi.ResPath

	// flLock guards fi.Md5, which the pack write stores as well, so the read
	// of it takes flLock; md5Mu guards the single-flight. Both are taken in
	// the order computeMd5 publishes, flLock -> md5Mu.
	p.flLock.RLock()
	p.md5Mu.Lock()
	if fi.Md5.IsValid() {
		hash := fi.Md5.String()
		p.md5Mu.Unlock()
		p.flLock.RUnlock()
		callback(hash, nil)
		return
	}
	if inflight, ok := p.md5Inflight[key]; ok {
		p.md5Inflight[key] = append(inflight, callback)
		p.md5Mu.Unlock()
		p.flLock.RUnlock()
		return
	}
	if p.md5Inflight == nil {
		p.md5Inflight = make(map[string][]func(string, error))
	}
	p.md5Inflight[key] = []func(string, error){callback}
	p.md5Mu.Unlock()
	p.flLock.RUnlock()

	go p.computeMd5(fi, key)
}

// computeMd5 reads fi once, publishes the hash under flLock and then invokes
// every callback that GetOrUpdateResourceMd5 queued for key. It runs in its
// own goroutine with the single-flight entry for key already registered.
func (p *Package) computeMd5(fi *structures.FileInfo, key string) {
	hash := md5.New()
	var err error

	switch {
	case p.mode == PackageModeUnpacked:
		l := p.log.WithField("filePath", fi.Path)
		file, openErr := os.Open(fi.Path)
		if openErr != nil {
			l.WithError(openErr).Error("can not open path to compute md5")
			err = dderrors.CausedBy(fmt.Errorf("failed to open %s for hashing", fi.Path), openErr)
			break
		}
		defer file.Close()

		if _, copyErr := io.Copy(hash, file); copyErr != nil {
			l.WithError(copyErr).Errorf("failed to read file")
			err = dderrors.CausedBy(fmt.Errorf("failed to read file"), copyErr)
		}
	case p.mode == PackageModePacked && p.pkgFile != nil:
		l := p.log.WithField("resPath", fi.ResPath)
		// readPackedFileFromPackage takes pkgFileMu around the seek+read
		// itself, so every reader of the shared file is serialised with
		// this computation
		data, readErr := p.readPackedFileFromPackage(p.pkgFile, fi)
		if readErr != nil {
			l.WithError(readErr).Errorf("failed to read packed file")
			err = dderrors.CausedBy(errors.New("failed to read packed file"), readErr)
			break
		}
		if _, writeErr := hash.Write(data); writeErr != nil {
			err = dderrors.CausedBy(errors.New("failed to write hash"), writeErr)
		}
	default:
		p.log.WithField("resPath", fi.ResPath).Error("Can not update hash for file info, unknown package state")
		err = dderrors.CausedBy(ErrPackageNotLoaded, errors.New("unknown package state"))
	}

	var (
		result  string
		waiters []func(string, error)
	)
	if err == nil {
		sum := hash.Sum(nil)
		// publish under both locks: flLock guards fi.Md5 for the rest of
		// the package, md5Mu keeps readers of fi.Md5 that only hold md5Mu
		// ordered with this write. Lock order flLock -> md5Mu.
		p.flLock.Lock()
		p.md5Mu.Lock()
		fi.Md5.UpdateWith(sum)
		result = fi.Md5.String()
		waiters = p.md5Inflight[key]
		delete(p.md5Inflight, key)
		p.md5Mu.Unlock()
		p.flLock.Unlock()
	} else {
		p.md5Mu.Lock()
		waiters = p.md5Inflight[key]
		delete(p.md5Inflight, key)
		p.md5Mu.Unlock()
	}

	// the callbacks run without any lock held
	for _, waiter := range waiters {
		waiter(result, err)
	}
}

type UpdateResourcesOptions struct {
	Md5  bool
	Size bool
}

func (p *Package) UpdateResources(options UpdateResourcesOptions, progressCallback func(p float64, max int64)) []error {
	numCPUs := runtime.NumCPU()
	sem := structures.NewSemaphore(numCPUs)
	wg := sync.WaitGroup{}

	// Only the snapshot of the file list is taken under flLock: the md5
	// single-flight publishes hashes under flLock (computeMd5), and the
	// workers below block waiting for those callbacks — holding flLock
	// across them would deadlock, because the publish could never acquire
	// the lock. The snapshot may go stale during the run; that is safe
	// because the workers hold *FileInfo pointers and the package replaces
	// file list entries rather than mutating them.
	p.flLock.Lock()
	fileList := p.fileList.AsSlice()
	p.flLock.Unlock()
	total := len(fileList)

	type result struct {
		md5Err  error
		sizeErr error
	}

	chResults := make(chan result, numCPUs*8)

	resourceErrors := []error{}
	resultsWg := sync.WaitGroup{}

	resultsWg.Go(func() {
		i := 0
		for {
			r, ok := <-chResults
			if !ok {
				return
			}
			if r.md5Err != nil {
				resourceErrors = append(resourceErrors, r.md5Err)
			}
			if r.sizeErr != nil {
				resourceErrors = append(resourceErrors, r.sizeErr)
			}
			i++
			progressCallback(float64(i)/float64(total), int64(total))
		}
	})

	for _, fi := range fileList {
		sem.Acquire()
		wg.Go(func() {
			defer sem.Release()
			innerWg := sync.WaitGroup{}
			var md5Err, sizeErr error
			if options.Md5 {
				innerWg.Add(1)
				p.GetOrUpdateResourceMd5(fi, func(_ string, err error) {
					defer innerWg.Done()
					md5Err = err
				})
			}
			if options.Size && p.mode == PackageModeUnpacked {
				innerWg.Add(1)
				fi.GetOrUpdateFilesystemSize(func(_ int64, err error) {
					defer innerWg.Done()
					sizeErr = err
				})
			}
			innerWg.Wait()
			chResults <- result{
				md5Err, sizeErr,
			}
		})
	}

	wg.Wait()
	close(chResults)
	resultsWg.Wait()
	return resourceErrors
}
