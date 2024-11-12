FROM node:lts-alpine AS ui_builder
RUN apk update && apk add git
ARG VER
WORKDIR /dashboard
COPY dashboard .
RUN npm i --ignore-scripts; npm run build; 


FROM golang:1.23 AS go_builder
ARG VER
WORKDIR /server
COPY pkg . 
COPY go.mod .
COPY go.sum .
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build


FROM scratch
LABEL org.opencontainers.image.title=warptail
LABEL org.opencontainers.image.source="https://github.com/robrotheram/warptail"
LABEL org.opencontainers.image.description='Tailscale network proxy'
LABEL org.opencontainers.image.documentation='https://github.com/robrotheram/warptail'
LABEL org.opencontainers.image.authors='robrotheram'
COPY --from=ui_builder /dashboard/dist /dashboard/dist
COPY --from=go_builder /server/warptail /go/bin/warptail
ENTRYPOINT ["/go/bin/warptail"]
