# PProf Web UI

This is a hacky experiment that serves the [Go pprof profiler web UI](https://github.com/google/pprof). You can upload pprof files then view them without installing anything. See [my blog post for some additional details](https://www.evanjones.ca/pprofweb.html).

Try it: https://pprofweb.evanjones.ca/


## Run Locally

docker build . --tag=pprofweb
docker run --rm -ti --publish=127.0.0.1:8080:8080 pprofweb

Open http://localhost:8080/


## Check that the container works

docker run --rm -ti --entrypoint=dot pprofweb
