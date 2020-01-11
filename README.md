# PProf Web UI

This is a total hack to upload pprof files and serve the UI. This avoids needing to install any tools.

Try it: https://pprofweb-kgdmaenclq-uc.a.run.app/


## Run Locally

docker build . --tag=pprofweb
docker run --rm -ti --publish=127.0.0.1:8080:8080 pprofweb

Open http://localhost:8080/


## Check that the container works

docker run --rm -ti --entrypoint=dot pprofweb
