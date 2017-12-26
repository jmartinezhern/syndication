FROM golang:1.9

WORKDIR /go/src/github.com/varddum/syndication
COPY ./config/syndication.toml /etc/syndication/config.toml
COPY . .

RUN curl -s https://glide.sh/get | sh
RUN glide install
Run go-wrapper install

EXPOSE 8080

CMD ["go-wrapper", "run"]
