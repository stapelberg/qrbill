# qrbill Go package

<img src="https://github.com/stapelberg/qrbill/raw/master/2020-06-25-qr-mary-jane.png" width="200" height="200" align="right">

The qrbill package implements generating QR-bill QR codes, which is a Swiss
🇨🇭 payment standard:

* [launch
announcement](https://www.six-group.com/en/newsroom/media-releases/2020/20200609-qr-bill-launch.html)

* [Specification: Swiss Implementation Guidelines
  QR-bill](https://www.paymentstandards.ch/dam/downloads/ig-qr-bill-en.pdf)
\
  (version 2.1 from 30 September 2019 at the time of writing)

These QR codes fill in details of a bank wire transfer, either all fields or
just some of them.

Recipients can scan the QR code with their mobile banking app, or with their
online banking web site’s mobile scan functionality, making it much easier to
send a wire transfer.

An example use case is for an association or club: you can send out payment
instructions to your members to collect membership fees (“Einzahlungsschein”).

## Validity

The [SIX QR-bill validation
portal](https://validation.iso-payments.ch/gp/qrrechnung/home) successfully
confirms that e.g. the code at the top right is valid.

## qrbill API (Example program)

To install the program:

```
go install github.com/stapelberg/qrbill/cmd/...
```

To run the program, run `qrbill-api` in the terminal. You will see an output like this:

```
% qrbill-api
2020/06/25 23:32:52 QR Bill generation URL: http://localhost:9933/qr?format=html
```

You can append the URL parameters on the left to customize the code. E.g.:

http://localhost:9933/qr?format=html&udname=Mary+Jane&udaddr1=Artikel+19b

When you open the URL in your web browser, you should see the following debug
output:

![](2020-06-25-format-html.jpg)

When you scan the resulting QR code with your eBanking app, you should see a
form like this:

<img src="https://github.com/stapelberg/qrbill/raw/master/2020-06-25-ebanking-app.jpg" height="300">

Once you are happy with the code parameters, change the `format` parameter from
`html` to `png`, e.g.:

http://localhost:9933/qr?format=png&udname=Mary+Jane&udaddr1=Artikel+19b
