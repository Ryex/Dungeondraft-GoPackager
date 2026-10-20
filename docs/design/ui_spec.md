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
