// Copyright 2013 Ryan Brown (ryan@ryanleebrown.com)

package main

import (
	"code.google.com/p/gopass"
	"coursync/browser"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	// Get user credentials.
	email := ""
	fmt.Print("Email: ")
	fmt.Scan(&email)
	password, err := gopass.GetPass("Password: ")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Allow user to ctrl+c.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(){
		for sig := range c {
			fmt.Println(sig)
			os.Exit(1)
		}
	}()

	fmt.Println("Signing in...")
	b := browser.New()
	err = b.SignIn(email, password)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Welcome, %s!\n", b.User.FullName)

	fmt.Println("Getting your course list...")
	topics, err := b.GetTopicList()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print topic/course list.
	fmt.Println()
	for i, topic := range topics {
		active := 0
		for _, course := range topic.Courses {
			if course.Active {
				active++
			}
		}
		fmt.Printf("%d. %s (%d of %d active)\n", i+1, topic.Name, active, len(topic.Courses))
	}
	fmt.Println()

	// Sync videos for each active course, two at a time.
	done := make(chan int)
	semaphore := make(chan int, 2)
	semaphore <- 1
	semaphore <- 1
	workers := 0
	for _, topic := range topics {
		for i, course := range topic.Courses {
			if !course.Active {
				fmt.Printf("%s (%d) is not active.\n", topic.Name, i+1)
				continue
			}
			fmt.Printf("Getting video list for %s (%d)...\n", topic.Name, i+1)
			err = b.AuthCourse(course)
			if err != nil {
				fmt.Println(err)
				continue
			}
			videos, err := b.GetVideoUrlList(course)
			if err != nil {
				fmt.Println(err)
				continue
			}
			dirname := fmt.Sprintf("%s-%s", topic.ShortName, course.Name)
			for j, video := range videos {
				workers++
				go func(dirname, video string, n, j int) {
					<-semaphore
					defer func(){
						semaphore <- 1
						done <- 1
					}()
					fmt.Printf("Syncing video %d of %d...\n", j+1, n)
					err := b.SyncVideo(dirname, video)
					if err != nil {
						if os.IsExist(err) {
							fmt.Printf("(video %d already exists)\n", j+1)
						} else {
							fmt.Println(err)
							return
						}
					}
				}(dirname, video, len(videos), j)
			}
		}
	}

	// Wait for the workers to finish.
	for i := 0; i < workers; i++ {
		<-done
	}

	fmt.Println("\nEnjoy the gift of knowledge!")
}
