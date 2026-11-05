# syntax=docker/dockerfile:1

# =========================================================================
# Tahap 1 — build: image besar berisi Go compiler, HANYA untuk compile.
# =========================================================================
FROM golang:1.26-alpine AS build
WORKDIR /src

# go.mod dan go.sum disalin lebih dulu, terpisah dari kode. Docker menyimpan
# hasil setiap instruksi sebagai layer. Selama dua file ini tidak berubah,
# layer "go mod download" dipakai ulang dari cache dan dependency tidak
# diunduh lagi — walaupun kode program berubah.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 menghasilkan binary statis yang tidak bergantung pada
# library C, sehingga dapat berjalan di image runtime yang sangat kecil.
# -trimpath membuang path folder komputer pembuat dari binary.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app .

# =========================================================================
# Tahap 2 — runtime: hanya binary, tanpa compiler dan tanpa source code.
# =========================================================================
FROM alpine:3.24

# Aplikasi TIDAK dijalankan sebagai root. Bila suatu saat ada celah
# keamanan di aplikasi, penyerang hanya mendapat hak user biasa.
RUN addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=build /out/app ./app

# Logger menulis ke logs/app.log, maka foldernya harus dapat ditulis user app.
RUN mkdir logs && chown app:app logs
USER app

EXPOSE 3000
ENTRYPOINT ["./app"]
CMD ["-role=api"]