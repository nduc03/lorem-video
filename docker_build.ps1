#!/usr/bin/env bash

# this script is bash compatible so just pretend this file is .sh if you are on linux/mac
# if on windows, just run the script as usual.

docker build -t forgejo.hs.lan/nduc/lorem-video -f Dockerfile .
docker build -t forgejo.hs.lan/nduc/lorem-video-pregen -f Dockerfile-pregen .

docker push forgejo.hs.lan/nduc/lorem-video
docker push forgejo.hs.lan/nduc/lorem-video-pregen
