# github release creation copied from instructions at:
# https://loads.pickle.me.uk/2015/08/22/easy-peasy-github-releases-for-go-projects-using-travis/
# https://web.archive.org/web/20161114025928/https://loads.pickle.me.uk/2015/08/22/easy-peasy-github-releases-for-go-projects-using-travis/

package = github.com/zellyn/diskii

.PHONY: release

release:
	mkdir -p release
	GOOS=linux GOARCH=amd64 go build -o release/diskii-linux-amd64 $(package)
	GOOS=darwin GOARCH=amd64 go build -o release/diskii-macos-amd64 $(package)
	GOOS=windows GOARCH=amd64 go build -o release/diskii-windows-amd64.exe $(package)
