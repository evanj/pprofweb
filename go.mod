module github.com/evanj/pprofweb

go 1.13

require (
	github.com/NYTimes/gziphandler v1.1.1
	// commit 427632fa3b1c fails as user nobody:
	// https://github.com/google/pprof/pull/542
	github.com/google/pprof v0.0.0-20200504201735-160c4290d1d8
	github.com/ianlancetaylor/demangle v0.0.0-20200524003926-2c5affb30a03 // indirect
)
