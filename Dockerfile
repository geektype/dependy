FROM cgr.dev/chainguard/wolfi-base:latest

WORKDIR /app

COPY ./bin/linux/dependy .


ENTRYPOINT [ "./dependy" ]
