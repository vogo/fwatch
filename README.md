# the design of file watch

## Background

I'd like to automatically operate all these files with a given suffix in a directory and all sub-directories.
These files not being updated for a long time should be excluded, 
but need to be included if them being updated in a later time.

## Features
- recursively list all file with given suffix in directory/sub-directories
- watch file creation event in directory/sub-directories
- watch file update event in directory/sub-directories

## Design

Structures:
- **watching directory list**: watch file/dir create event, add file to operation file list, send it wait operation channel.
- **watching file list**: watch write event for files, move updated file to active file list, send it to active channel, and stop watch.
- **active file list**: schedule checkFiles whether not updated for a long time, if yes then send it inactive channel, and add it to watching file list.
- **active channel**: for active files notification.
- **inactive channel**: for inactive files notification.
- **file active deadline duration**: a duration if a file not being updated in, then move it to watching file list.

library:
- https://github.com/vogo/fsnotify

