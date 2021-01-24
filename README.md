1. cd $GOPATH/src
2. git clone https://github.com/MaxBear/HttpGetter.git
3. cd HttpGetter/
4. run "go build" to genreate HttpGetter binary
5. run "./HttpGetter --help" to show usage
6. run "./HttpGetter -f urls.txt" to run throttled http get
7. after HttpGetter, you should see output stored as url_[n].html where n is the index of url in input file, eg. urls.txt
