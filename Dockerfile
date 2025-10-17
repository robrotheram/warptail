FROM node:latest AS ui_builder
ARG VER
COPY dashboard /dashboard
WORKDIR /dashboard
RUN npm i; npm run build; 


FROM golang:1.25 AS go_builder
ARG VER
WORKDIR /server
COPY --from=ui_builder /dashboard/dist /server/dashboard/dist
COPY pkg pkg 
COPY go.mod go.mod 
COPY go.sum go.sum
COPY main.go main.go
RUN CGO_ENABLED=0 GOOS=linux go build


FROM scratch
LABEL org.opencontainers.image.title=warptail
LABEL org.opencontainers.image.source="https://github.com/robrotheram/warptail"
LABEL org.opencontainers.image.description='Tailscale network proxy'
LABEL org.opencontainers.image.documentation='https://github.com/robrotheram/warptail'
LABEL org.opencontainers.image.authors='robrotheram'
COPY --from=go_builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go_builder /server/warptail /go/bin/warptail
ENTRYPOINT ["/go/bin/warptail"]
