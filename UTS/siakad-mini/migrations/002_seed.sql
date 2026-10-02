-- Seeder 002: data awal SIAKAD Mini
-- 1 admin, 20 mahasiswa, 11 mata kuliah (syarat: minimal 10).
-- Password disimpan sebagai hash bcrypt (cost 12):
--   admin  : admin1234
--   mahasiswa: NIM-nya masing-masing.
-- Hash dihasilkan oleh go run ./cmd/hashgen. Jalankan seeder ini
-- sekali saja pada database yang masih kosong.

-- ===== users =====

INSERT INTO users (email, password, role) VALUES
    ('admin@siakad.test', '$2a$12$0McUqaybmBIZfDktAwUjIuePQzh3FxwHXyZsFH1LF.AOMbTV8872O', 'admin');

INSERT INTO users (email, password, role) VALUES
    ('000434241080@siakad.test', '$2a$12$kdCKSXvVOy3TIg9IMX5/VOfysUhKOEpjMhb9HKHJXTcVFONb.qZ3e', 'mahasiswa'),
    ('000434241081@siakad.test', '$2a$12$OmAIWmeiYgRFWS6SC22GeuO030g5BJiXx4r16o8JDckG9mmY/it9O', 'mahasiswa'),
    ('000434241082@siakad.test', '$2a$12$QYyX0sMQJRkqweASxikXzeBQG/SfmhquLA9Y6VJWjiRzywyVtHHCC', 'mahasiswa'),
    ('000434241083@siakad.test', '$2a$12$SlJZe4HaRiqb1pBDAN2xFeg0ShKTOJlaQMImZGwlNflHslPkC3vO2', 'mahasiswa'),
    ('000434241084@siakad.test', '$2a$12$5En6ZVUpSmaGq2Tw6w8adufrcZ/.Y2pCtJE11Jy/qU1LYYljopuFq', 'mahasiswa'),
    ('000434241085@siakad.test', '$2a$12$h3lKf1T0klBa0lB2vys97e/sI4XJ2hUdwKwSIiqzpBnSibR9dxwGq', 'mahasiswa'),
    ('000434241086@siakad.test', '$2a$12$fYMVEfWoGCnInUniZy6CLeMTmEythOe/RYa5Q5all.XuSVCvR43tu', 'mahasiswa'),
    ('000434241087@siakad.test', '$2a$12$b7/ArxZN5llUbA70m/maKeXmVSE.jw9xd2UCiKciX.DFoE1Lhzbt2', 'mahasiswa'),
    ('000434241088@siakad.test', '$2a$12$Q9wAA4UoREVZkj.viCbC1OAA67mIR7gAOw1vojNdvavIUFJMyXMi.', 'mahasiswa'),
    ('000434241089@siakad.test', '$2a$12$L0.vxedAyUpwHCmlBhiNi.054XRFOuuTht2kkByUYt.54vXgraHQS', 'mahasiswa'),
    ('000434241090@siakad.test', '$2a$12$PZRoWEflPSepYM6yXW.hLOwZLK2ogVSwcJqyB3BkIuN1eKQ3GrFfu', 'mahasiswa'),
    ('000434241091@siakad.test', '$2a$12$VRSWpqNsoZ1Uvl/oVoXgO.k6JurwhCH.csVwIk.4tgVxij8EB.sda', 'mahasiswa'),
    ('000434241092@siakad.test', '$2a$12$/vvzo6qoSRh3tL1t4wmCS.Oy5r7doFh55R.78vrt3nagRPHhpp7kW', 'mahasiswa'),
    ('000434241093@siakad.test', '$2a$12$sOv5t3o3.8CXE2NHtipKe.8DF5KLOjk/JuoNZMbRO4G/4XtJRQEQW', 'mahasiswa'),
    ('000434241094@siakad.test', '$2a$12$zDr2GwrJWjQO5l5rrJqoT.TPrrI.jI.Fk56b4TfPpylcy98.D7jey', 'mahasiswa'),
    ('000434241095@siakad.test', '$2a$12$3h0S3g/7.xyfPQvf5.mtCO4Lus4XAfEShiJ.WARo3./udHNDWZPCe', 'mahasiswa'),
    ('000434241096@siakad.test', '$2a$12$b01W/kIm32e/na0Rnb/SLO49QsMRv1DUg6Hm7SntuVMOBpYGw0vUC', 'mahasiswa'),
    ('000434241097@siakad.test', '$2a$12$VFzy2uyl8m.Hwy78s8meQefOnOFcMbVogR0aUxsvwm/gm8cr548PO', 'mahasiswa'),
    ('000434241098@siakad.test', '$2a$12$7Uq2Vg0JGBTTB5vEtHhaZ.s2.mpMa.jL7cXEbZGHMgnQLJkHhCuyC', 'mahasiswa'),
    ('000434241099@siakad.test', '$2a$12$pnzITP7nxIYBIgyIDaDGD./GcuMCV6ea/CScgkSeAhj5tnmBVmA3e', 'mahasiswa');

-- ===== students =====

INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
SELECT u.id, t.nim, t.nama, t.prodi, t.angkatan, t.ipk
FROM (VALUES
    ('000434241080', 'Putri Atika Mahardika Dewi', 'Sistem Informasi', 2024, 3.65),
    ('000434241081', 'Muhammad Fachriditya', 'Sistem Informasi', 2024, 3.12),
    ('000434241082', 'Naily Nugraheni', 'Teknik Informatika', 2023, 2.78),
    ('000434241083', 'Muhammad Ikhsanudin Arsalan', 'Sistem Informasi', 2024, 3.87),
    ('000434241084', 'Vito Aditya', 'Sistem Informasi', 2024, 2.95),
    ('000434241085', 'Muhammad Rizki Ibrahim', 'Sistem Informasi', 2024, 3.40),
    ('000434241086', 'Allaedine Zidane', 'Teknik Informatika', 2023, 2.31),
    ('000434241087', 'Hidayatullah Sukma Dewi', 'Sistem Informasi', 2024, 3.55),
    ('000434241088', 'Valerina Dzakiyya Salsabila', 'Sistem Informasi', 2024, 2.60),
    ('000434241089', 'Ananda Farrel A. S.', 'Sistem Informasi', 2023, 3.05),
    ('000434241090', 'Faatin Sausan Firdaus', 'Sistem Informasi', 2024, 3.21),
    ('000434241091', 'Ferdyano Surya Dynata', 'Teknik Informatika', 2024, 2.85),
    ('000434241092', 'Lailia Trihapsari Subagyo', 'Sistem Informasi', 2024, 3.92),
    ('000434241093', 'Grachya Ayuma Putri Prissyllia', 'Sistem Informasi', 2024, 2.45),
    ('000434241094', 'Adnan Fadholi', 'Teknik Informatika', 2023, 3.30),
    ('000434241095', 'Faza Nazzala Anindita', 'Sistem Informasi', 2024, 2.72),
    ('000434241096', 'Raihan Zulfa Kamal', 'Sistem Informasi', 2024, 3.75),
    ('000434241097', 'Fahmi Rizki Yuviyanto', 'Teknik Informatika', 2024, 2.15),
    ('000434241098', 'Maulidina Naurah Salsabila', 'Sistem Informasi', 2024, 3.02),
    ('000434241099', 'Muhammad Fadhil Ilyas', 'Sistem Informasi', 2023, 2.90)
) AS t(nim, nama, prodi, angkatan, ipk)
JOIN users u ON u.email = t.nim || '@siakad.test';

-- ===== courses =====

INSERT INTO courses (kode_mk, nama_mk, sks, semester, kuota) VALUES
    ('AGI401', 'Agama Islam II', 2, 3, 40),
    ('SIC307', 'Pembelajaran Mesin', 2, 5, 35),
    ('SIC308', 'Pembelajaran Mesin (Praktikum)', 1, 5, 3),
    ('SII329', 'Design Thinking', 2, 5, 40),
    ('SIJ304', 'Keamanan Cyber', 2, 5, 35),
    ('SIJ305', 'Keamanan Cyber (Praktikum)', 1, 5, 30),
    ('SIP374', 'Pemrograman Backend Lanjut', 1, 5, 35),
    ('SIP375', 'Pemrograman Backend Lanjut (Praktikum)', 2, 5, 30),
    ('SIR302', 'Proyek 1 (Analisa dan Desain PL)', 2, 5, 30),
    ('SIR303', 'Jaminan Kualitas Perangkat Lunak (Quality Assurance)', 2, 5, 25),
    ('SIR307', 'Kewirausahaan Bidang IT', 2, 5, 40);
