FROM alpine:3.8 AS build

RUN apk add --no-cache curl git build-base gcc linux-headers patch
RUN git clone --recursive https://github.com/iotivity/iotivity-lite.git
RUN (cd iotivity-lite && git reset --hard 28ae13dbe13ce673789f19554dc6b72881e3880f)
RUN make -C /iotivity-lite/port/linux CLOUD=1 SECURE=1 DEBUG=1 cloud_linux

FROM alpine:3.8 AS service
COPY --from=build /iotivity-lite/port/linux/cloud_linux /usr/local/bin/service
COPY ./cloud_linux_creds ./cloud_linux_creds
ENTRYPOINT ["/usr/local/bin/service"]