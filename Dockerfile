FROM gcr.io/distroless/base-debian10:nonroot
COPY accord-server /usr/local/bin/
ENTRYPOINT ["accord-server"]
EXPOSE 7475
