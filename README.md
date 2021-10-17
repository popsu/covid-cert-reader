# Covid-cert reader

Decoder for EU Digital COVID Certificate (EUDCC) QR code data. Does **not** verify the signature. Created for learning purpose

## Usage

```bash
go install github.com/popsu/covid-cert-reader@v0.0.1

curl -s https://raw.githubusercontent.com/eu-digital-green-certificates/dgc-testdata/main/FI/png/10.png > testimage.png

covid-cert-reader testimage.png
```

## Links

- [spec](https://github.com/ehn-dcc-development/hcert-spec)
- [Blog post + Python implementation](https://gir.st/blog/greenpass.html)
- [CBOR spec](https://cbor.io/)
- [Digital Covid Certificate Schema](https://github.com/ehn-dcc-development/ehn-dcc-schema)
- [More complete Go solution](https://github.com/stapelberg/coronaqr)
