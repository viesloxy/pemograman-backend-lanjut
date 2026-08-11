package main

import "fmt"

func main() {
	var nama string = "Vito Aditya"
	var umur int = 20
	var ipk float64 = 3.64
	var mahasiswaAktif bool = true
	hobi := []string{"ngodonf,", " larp,", " gaming"}

	fmt.Println("Data diri mahasiswa")
	fmt.Println("nama         :", nama)
	fmt.Println("umur         :", umur)
	fmt.Println("ipk          :", ipk)
	fmt.Println("aktif kuliah :", mahasiswaAktif)
	fmt.Println("hobi         :", hobi)

	dataMahasiswa := make(map[string]int)

	dataMahasiswa["Vito"] = 90
	dataMahasiswa["Sari"] = 85
	dataMahasiswa["Budi"] = 78
	dataMahasiswa["Ani"] = 92
	fmt.Println("\ndata nilai mahasiswa")
	fmt.Println("setelah menambah 4 mahasiswa:", dataMahasiswa)

	nilaiVito, ada := dataMahasiswa["Vito"]
	if ada {
		fmt.Println("nilai vito ditemukan:", nilaiVito)
	} else {
		fmt.Println("data vito belum ada")
	}

	nilaiRudi, ada := dataMahasiswa["Rudi"]
	if ada {
		fmt.Println("nilai rudi ditemukan:", nilaiRudi)
	} else {
		fmt.Println("data rudi belum ada di map")
	}

	delete(dataMahasiswa, "Budi")
	fmt.Println("setelah menghapus Budi:", dataMahasiswa)

	fmt.Println("\ndaftar seluruh mahasiswa")
	for nama, nilai := range dataMahasiswa {
		fmt.Printf("- %s: %d\n", nama, nilai)
	}
}
