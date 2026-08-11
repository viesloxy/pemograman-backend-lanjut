package main

import "fmt"

func swap(a, b *int) {
	*a, *b = *b, *a
}

func updateSlice(s *[]string, newItem string) {
	*s = append(*s, newItem)
}

func tambahDenganValue(x int) {
	x = x + 100
	fmt.Println("  di dalam function value, x =", x)
}

func tambahDenganPointer(x *int) {
	*x = *x + 100
	fmt.Println("  di dalam function pointer, *x =", *x)
}

func main() {
	fmt.Println("tes swap(a, b *int)")
	a := 10
	b := 20
	fmt.Println("sebelum swap: a =", a, ", b =", b)
	swap(&a, &b)
	fmt.Println("sesudah swap: a =", a, ", b =", b)

	fmt.Println("\ntes updateSlice(s *[]string, newItem string)")
	buah := []string{"apel", "mangga"}
	fmt.Println("sebelum update:", buah, "len =", len(buah))
	updateSlice(&buah, "jeruk")
	updateSlice(&buah, "pisang")
	fmt.Println("sesudah update:", buah, "len =", len(buah))

	fmt.Println("\nperbandingan pass by value vs pass by pointer")
	angka := 50
	fmt.Println("nilai awal angka:", angka)

	fmt.Println("- pass by value -")
	tambahDenganValue(angka)
	fmt.Println("setelah tambahDenganValue, angka =", angka,
		"(tidak berubah karena function hanya menerima salinan)")

	fmt.Println("- pass by pointer -")
	tambahDenganPointer(&angka)
	fmt.Println("setelah tambahDenganPointer, angka =", angka,
		"(berubah karena function mengubah lewat alamat)")
}
