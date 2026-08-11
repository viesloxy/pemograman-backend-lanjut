package main

import "fmt"

type Student struct {
	ID       int
	Name     string
	Grade    float64
	IsActive bool
}

func (s Student) GetInfo() string {
	status := "tidak aktif"
	if s.IsActive {
		status = "aktif"
	}
	return fmt.Sprintf("id: %d | nama: %s | nilai: %.2f | status: %s",
		s.ID, s.Name, s.Grade, status)
}

func (s *Student) UpdateGrade(grade float64) {
	s.Grade = grade
}

func (s *Student) Activate() {
	s.IsActive = true
}

func (s *Student) Deactivate() {
	s.IsActive = false
}

func main() {
	mhs := Student{
		ID:    1,
		Name:  "Vito Aditya",
		Grade: 80.5,
	}

	fmt.Println("data awal")
	fmt.Println(mhs.GetInfo())

	fmt.Println("\nsetelah mhs.Activate()")
	mhs.Activate()
	fmt.Println(mhs.GetInfo())

	fmt.Println("\nsetelah mhs.UpdateGrade(95.75)")
	mhs.UpdateGrade(95.75)
	fmt.Println(mhs.GetInfo())

	fmt.Println("\nsetelah mhs.Deactivate()")
	mhs.Deactivate()
	fmt.Println(mhs.GetInfo())
}
