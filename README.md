# LockedArchive [![Apache-2.0 License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](https://github.com/jonathan-robertson/lockedarchive/blob/master/LICENSE)

This project is meant for storing/archiving family information in a safe way (locally encrypted): Documents, Tax forms, scans of important licenses, etc.

## Development

If this project interests you, star/watch it and feel free get involved.

1. `go get -u github.com/jonathan-robertson/lockedarchive`
1. Install Glide for dependency management
   - macOS
     1. If you don't already have it, install Homebrew according to [this guide](https://brew.sh)
     1. If you don't already have it, install Glide for dependency management `brew install glide`
     1. Navigate to $GOPATH/src/github.com/jonathan-robertson/lockedarchive
     1. Update dependencies with `glide update --strip-vendor` or `glide u -v` (alias)
        - `--strip-vendor` because I've included the dependencies/vendors of each project within my own

## Documentation Licensing

The content of this document and all other documents within this repository are licensed under the Creative Commons Attribution 3.0 License.
