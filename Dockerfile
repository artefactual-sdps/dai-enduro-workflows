# syntax = docker/dockerfile:1.4

ARG GO_VERSION

FROM golang:${GO_VERSION}-alpine AS build-go
WORKDIR /src
ENV CGO_ENABLED=0
COPY --link go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY --link . .

FROM build-go AS build-dai-enduro-worker
ARG VERSION_PATH
ARG VERSION_LONG
ARG VERSION_SHORT
ARG VERSION_GIT_HASH
ARG STRIP=1
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	ldflags="-X '${VERSION_PATH}.Long=${VERSION_LONG}' -X '${VERSION_PATH}.Short=${VERSION_SHORT}' -X '${VERSION_PATH}.GitCommit=${VERSION_GIT_HASH}'" && \
	if [ "$STRIP" = "1" ]; then ldflags="-s $ldflags"; fi && \
	go build \
	-trimpath \
	-ldflags="$ldflags" \
	-o /out/dai-enduro-worker \
	./cmd/worker

FROM alpine:3.22 AS base
ARG USER_ID=1000
ARG GROUP_ID=1000
RUN addgroup -g ${GROUP_ID} -S enduro
RUN adduser -u ${USER_ID} -S -D enduro enduro
USER enduro
RUN mkdir /home/enduro/shared

FROM alpine:3.22 AS csv-validator
ARG CSV_VALIDATOR_VERSION=1.4.3
ARG CSV_VALIDATOR_SHA256=8b621c38f2542803fae5fd56965383993b7ae02d203f7832c4ec0a563877a838
RUN apk add --no-cache curl unzip
RUN curl -fsSL -o /tmp/csv-validator.zip \
		"https://repo1.maven.org/maven2/uk/gov/nationalarchives/csv-validator-distribution/${CSV_VALIDATOR_VERSION}/csv-validator-distribution-${CSV_VALIDATOR_VERSION}-bin.zip" && \
	echo "${CSV_VALIDATOR_SHA256}  /tmp/csv-validator.zip" | sha256sum -c - && \
	mkdir -p /opt/csv-validator && \
	unzip -q /tmp/csv-validator.zip -d /opt/csv-validator && \
	chmod +x /opt/csv-validator/csv-validator-cmd && \
	rm /tmp/csv-validator.zip

FROM base AS dai-enduro-worker
USER root
# csv-validator 1.4.3 requires Java 21 and bash (launch script shebang).
RUN apk add --no-cache openjdk21-jre-headless bash
COPY --from=csv-validator --link /opt/csv-validator /opt/csv-validator
RUN ln -s /opt/csv-validator/csv-validator-cmd /usr/local/bin/csv-validator-cmd
USER enduro
COPY --from=build-dai-enduro-worker --link /out/dai-enduro-worker /home/enduro/bin/dai-enduro-worker
CMD ["/home/enduro/bin/dai-enduro-worker"]
