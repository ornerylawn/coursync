package main

import (
	"coursync/browser"
	"fmt"
	"code.google.com/p/gopass"
	"io"
	"os"
	"time"
)

func main() {
	// Get user credentials.
	email := ""
	fmt.Print("Email: ")
	fmt.Scan(&email)
	password, err := gopass.GetPass("Password: ")
	if err != nil {
		panic(err)
	}

	fmt.Println("Signing in...")

	b := browser.New()
	err = b.SignIn(email, password)
	if err != nil {
		panic(err)
	}

	fmt.Println("Pausing for human-like behavior...")
	time.Sleep(browser.PauseDuration)

	fmt.Println("Obtaining your course list...")
	courses, err := b.GetCourseList()
	if err != nil {
		panic(err)
	}

	fmt.Println()
	for i, v := range courses {
		fmt.Printf("%d. %s\n", i+1, v)
	}
	fmt.Println()

	for _, course := range courses {
		fmt.Println("Pausing for human-like behavior...")
		time.Sleep(browser.PauseDuration)

		fmt.Printf("Signing into %s...\n", course)
		err = b.AuthCourse(course)
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println("Pausing for human-like behavior...")
		time.Sleep(browser.PauseDuration)
		fmt.Printf("Obtaining video list for %s...\n", course)
		videos, err := b.GetVideoList(course)
		if err != nil {
			fmt.Println(err)
			continue
		}

		for i, video := range videos {
			fmt.Println("Pausing for human-like behavior...")
			time.Sleep(browser.PauseDuration)

			fmt.Printf("Downloading video %d of %d...", i, len(videos))
			r, err := b.DownloadVideo(course, video)
			if err != nil {
				fmt.Println(err)
				continue
			}
			f, err := os.OpenFile(course + video + ".mp4", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
			if err != nil {
				fmt.Println(err)
				continue
			}
			io.Copy(f, r)
		}
	}
}
