# Neon Database CRUD API

API Go untuk mengelola tabel `students`, `teachers`, dan `staff` yang tersimpan di Neon Database.

## Fitur

- List semua data
- Insert data baru
- Update data berdasarkan id
- Delete data berdasarkan id
- Seed data awal sesuai request Anda

## Data awal yang di-seed

```sql
INSERT INTO students (nama, umur, tanggal)
VALUES
('Lita', 20, '2006-05-12'),
('Budi', 21, '2005-08-20'),
('Sinta', 19, '2007-01-15');

INSERT INTO teachers (nama, umur, tanggal)
VALUES
('Pak Andi', 40, '1986-03-10'),
('Bu Rina', 35, '1991-07-22');

INSERT INTO staff (nama, umur, tanggal)
VALUES
('Dina', 28, '1998-02-14'),
('Rudi', 32, '1994-11-05');
```

## Persiapan env

1. Salin file `.env.example` menjadi `.env`
2. Isi `DATABASE_URL` dengan string koneksi Neon Database
3. Jalankan aplikasi

## Menjalankan aplikasi

```bash
go mod tidy
go run .
```

Aplikasi akan berjalan di `http://localhost:8080`.

## Endpoint utama

- `GET /health`
- `GET /api/students`
- `POST /api/students`
- `PUT /api/students/{id}`
- `DELETE /api/students/{id}`
- `GET /api/teachers`
- `POST /api/teachers`
- `PUT /api/teachers/{id}`
- `DELETE /api/teachers/{id}`
- `GET /api/staff`
- `POST /api/staff`
- `PUT /api/staff/{id}`
- `DELETE /api/staff/{id}`

## Contoh request

```bash
curl http://localhost:8080/api/students
curl -X POST http://localhost:8080/api/students \
  -H "Content-Type: application/json" \
  -d '{"nama":"Lita","umur":20,"tanggal":"2006-05-12"}'
```

## Deploy ke Render

- Buat Web Service baru di Render
- Hubungkan repository GitHub
- Set build command: `go build ./...`
- Set start command: `./week3_lita` atau `./<module-name>` sesuai hasil build
- Tambahkan environment variable `DATABASE_URL`

## Catatan

Untuk deploy ke Vercel, biasanya digunakan fungsi serverless atau routing khusus. Untuk backend database PostgreSQL, Render lebih cocok dan lebih sederhana untuk aplikasi Go ini.
