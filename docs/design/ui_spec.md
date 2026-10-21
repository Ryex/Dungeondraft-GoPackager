# UI Spec

list of declarative conditions to translate into tests.
sub lists represent conditions that should be read as: when <parent arm> then <all children>

- when the main window is shown
  - there is a welcome header at the top
    - with a theme toggle on the left
      - with a language select dropdown to the right of the theme toggle
    - with a title and subtitle in the center
    - with a "about" ion button to the right
      - with a github icon button to the right of "about"
        - with a "info" icon button to the right of "github"
  - with a path input section below the welcome header
    - with a path input entry to the left
    - with a "Open Pack" button to the right of the path entry
    - with a "Open Folder" button to the right of the "Open Pack" button
  - with a content section on the bottom

- when the app has just started and no pack has been opened yet
  - the main content shows main.defaultText "Enter a path to get started." ...

- when a pack folder is opened
  - the main content shows activity content until it loads
  - when the pack is loaded
    - the main content shows pack content
      - with a button panel on top
        - with a output path entry
        - with a browse button
        - with a thumbnail generate button
        - with a edit pack info button
        - with a generate tags button
        - with a edit tag sets button
        - with an overwrite check box
        - with a package button
      - with a split pane below the button pane
        - with a package resource tree on the left
        - with a resource pane on the right

- when a packed pack file is opened
  - the main content shows activity content until it loads
  - when the pack is loaded
    - the main content shows pack content
      - with a button panel on top
        - with a output path entry
        - with a browse button
        - with a package information button
        - with a view tag sets button
        - with an overwrite check box
        - with a rip textures check box
        - with an extract thumbnails check box
        - with an extract button
      - with a split pane below the button pane
        - with a package resource tree on the left
        - with a resource pane on the right

- when a resource is selected in the resource tree
  - the resource pane shows the information of the resource
    - with a file preview tab
      - an image resource shows the image
        - and the thumbnail toggle
      - a text resource shows the text
        - unless the file is too big, then the text is not shown
    - with a tag tab when the resource is taggable
    - with a metadata tab when the resource has metadata

- when more than one resource is selected
  - the resource pane shows the bulk operations pane
    - with a tags tab
      - with a "Bulk Tag Operations" header
      - with a list of the selected resources
        - with the icon, the file name and the path of every resource
        - with a button to deselect the resource
      - with a list of the tags that all the resources have in common
        - with a button to remove the tag from all the resources
      - with an entry to add a tag to all the resources

- when the filter of the resource tree is changed
  - a name filter shows the resources with a matching name only
  - a tag filter shows the resources with a matching tag only
  - a path filter shows the resources with a matching path only
  - a kind filter shows the resources of that kind only
  - an empty filter shows all the resources again
