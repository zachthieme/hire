package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"hire/internal/api"
	"hire/internal/models"
	"hire/internal/store"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	// Run migrations
	mig, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrate up: %v", err)
	}

	s, err := store.New(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()

	// Clean existing data
	s.DB().Exec("TRUNCATE competency_ratings, notifications, feedback, interviews, interview_loops, competencies, candidates, users RESTART IDENTITY CASCADE")

	// Admin user
	adminHash, _ := api.HashPassword("admin")
	admin := &models.User{Email: "admin@hire.demo", Name: "Admin User", PasswordHash: adminHash, Role: "admin"}
	s.CreateUser(ctx, admin)

	// Scheduler
	schedHash, _ := api.HashPassword("scheduler")
	sched := &models.User{Email: "scheduler@hire.demo", Name: "Sarah Scheduler", PasswordHash: schedHash, Role: "scheduler"}
	s.CreateUser(ctx, sched)

	// Interviewers
	ivHash, _ := api.HashPassword("interviewer")
	alice := &models.User{Email: "alice@hire.demo", Name: "Alice Engineer", PasswordHash: ivHash, Role: "interviewer"}
	s.CreateUser(ctx, alice)
	bob := &models.User{Email: "bob@hire.demo", Name: "Bob Designer", PasswordHash: ivHash, Role: "interviewer"}
	s.CreateUser(ctx, bob)
	carol := &models.User{Email: "carol@hire.demo", Name: "Carol Manager", PasswordHash: ivHash, Role: "interviewer"}
	s.CreateUser(ctx, carol)
	dave := &models.User{Email: "dave@hire.demo", Name: "Dave Architect", PasswordHash: ivHash, Role: "interviewer"}
	s.CreateUser(ctx, dave)

	// Competencies
	s.CreateCompetency(ctx, &models.Competency{Name: "Problem Solving", RatingType: "levels", RatingsJSON: `["Learning","Owning","Advising"]`})
	s.CreateCompetency(ctx, &models.Competency{Name: "Communication", RatingType: "levels", RatingsJSON: `["Learning","Owning","Advising"]`})
	s.CreateCompetency(ctx, &models.Competency{Name: "Technical Depth", RatingType: "stars", RatingsJSON: `{"min":1,"max":5}`})
	s.CreateCompetency(ctx, &models.Competency{Name: "Culture Fit", RatingType: "stars", RatingsJSON: `{"min":1,"max":5}`})

	// Candidates
	jane := &models.Candidate{Name: "Jane Smith", Email: "jane@example.com", ResumeURL: "https://example.com/resume/jane", Status: "active"}
	s.CreateCandidate(ctx, jane)
	mike := &models.Candidate{Name: "Mike Johnson", Email: "mike@example.com", ResumeURL: "https://example.com/resume/mike", Status: "active"}
	s.CreateCandidate(ctx, mike)

	// Interview loop for Jane
	loop1 := &models.InterviewLoop{CandidateID: jane.ID, Status: "active", CreatedBy: sched.ID}
	s.CreateLoop(ctx, loop1)

	tomorrow := time.Now().Add(24 * time.Hour)
	s.CreateInterview(ctx, &models.Interview{LoopID: loop1.ID, InterviewerID: alice.ID, FocusArea: "Coding", ScheduledAt: tomorrow, VideoLink: "https://meet.example.com/jane-coding", NotesForInterviewer: "Focus on data structures and algorithms", Status: "pending"})
	s.CreateInterview(ctx, &models.Interview{LoopID: loop1.ID, InterviewerID: bob.ID, FocusArea: "System Design", ScheduledAt: tomorrow.Add(time.Hour), VideoLink: "https://meet.example.com/jane-design", NotesForInterviewer: "Distributed systems focus", Status: "pending"})
	s.CreateInterview(ctx, &models.Interview{LoopID: loop1.ID, InterviewerID: carol.ID, FocusArea: "Behavioral", ScheduledAt: tomorrow.Add(2 * time.Hour), VideoLink: "https://meet.example.com/jane-behavioral", NotesForInterviewer: "Leadership and teamwork", Status: "pending"})
	s.CreateInterview(ctx, &models.Interview{LoopID: loop1.ID, InterviewerID: dave.ID, FocusArea: "Architecture", ScheduledAt: tomorrow.Add(3 * time.Hour), VideoLink: "https://meet.example.com/jane-arch", NotesForInterviewer: "API design and scalability", Status: "pending"})

	// Interview loop for Mike (scheduling phase)
	loop2 := &models.InterviewLoop{CandidateID: mike.ID, Status: "scheduling", CreatedBy: sched.ID}
	s.CreateLoop(ctx, loop2)
	s.CreateInterview(ctx, &models.Interview{LoopID: loop2.ID, InterviewerID: alice.ID, FocusArea: "Coding", ScheduledAt: tomorrow.Add(48 * time.Hour), VideoLink: "https://meet.example.com/mike-coding", Status: "pending"})

	// Notifications
	s.CreateNotification(ctx, &models.Notification{UserID: alice.ID, Message: "You've been assigned a Coding interview", Link: "/interviews/1"})
	s.CreateNotification(ctx, &models.Notification{UserID: bob.ID, Message: "You've been assigned a System Design interview", Link: "/interviews/2"})
	s.CreateNotification(ctx, &models.Notification{UserID: carol.ID, Message: "You've been assigned a Behavioral interview", Link: "/interviews/3"})
	s.CreateNotification(ctx, &models.Notification{UserID: dave.ID, Message: "You've been assigned an Architecture interview", Link: "/interviews/4"})

	fmt.Println("Seed data created successfully!")
	fmt.Println()
	fmt.Println("Demo accounts:")
	fmt.Println("  admin@hire.demo      / admin        (Admin)")
	fmt.Println("  scheduler@hire.demo  / scheduler    (Scheduler)")
	fmt.Println("  alice@hire.demo      / interviewer  (Interviewer)")
	fmt.Println("  bob@hire.demo        / interviewer  (Interviewer)")
	fmt.Println("  carol@hire.demo      / interviewer  (Interviewer)")
	fmt.Println("  dave@hire.demo       / interviewer  (Interviewer)")
}
