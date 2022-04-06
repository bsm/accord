FROM gcr.io/distroless/base-debian11:nonroot
COPY accord-server /usr/local/bin/
ENTRYPOINT ["accord-server"]
EXPOSE 7475
