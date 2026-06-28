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

	// Clean existing data (new-model tables)
	s.DB().Exec("TRUNCATE competency_ratings, notifications, feedback, stage_interviewers, stages, applications, jobs, competencies, candidates, users RESTART IDENTITY CASCADE")

	// Users
	adminHash, _ := api.HashPassword("admin")
	admin := &models.User{Email: "admin@hire.demo", Name: "Admin User", PasswordHash: adminHash, Role: "admin"}
	s.CreateUser(ctx, admin)

	schedHash, _ := api.HashPassword("scheduler")
	sched := &models.User{Email: "scheduler@hire.demo", Name: "Sarah Scheduler", PasswordHash: schedHash, Role: "scheduler"}
	s.CreateUser(ctx, sched)

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
	problemSolving := &models.Competency{Name: "Problem Solving", RatingType: "levels", RatingsJSON: `["Learning","Owning","Advising"]`}
	s.CreateCompetency(ctx, problemSolving)
	communication := &models.Competency{Name: "Communication", RatingType: "levels", RatingsJSON: `["Learning","Owning","Advising"]`}
	s.CreateCompetency(ctx, communication)
	s.CreateCompetency(ctx, &models.Competency{Name: "Technical Depth", RatingType: "stars", RatingsJSON: `{"min":1,"max":5}`})
	s.CreateCompetency(ctx, &models.Competency{Name: "Culture Fit", RatingType: "stars", RatingsJSON: `{"min":1,"max":5}`})

	// Candidates (no status field anymore)
	jane := &models.Candidate{Name: "Jane Smith", Email: "jane@example.com", ResumeURL: "https://example.com/resume/jane"}
	s.CreateCandidate(ctx, jane)
	mike := &models.Candidate{Name: "Mike Johnson", Email: "mike@example.com", ResumeURL: "https://example.com/resume/mike"}
	s.CreateCandidate(ctx, mike)

	// Jobs
	backend := &models.Job{Title: "Senior Backend Engineer", Description: "Design and build our core APIs and data systems.", HiringManager: "Priya Patel", Status: "open", CreatedBy: sched.ID}
	s.CreateJob(ctx, backend)
	designer := &models.Job{Title: "Product Designer", Description: "Own the end-to-end design of new product surfaces.", HiringManager: "Sam Lee", Status: "open", CreatedBy: sched.ID}
	s.CreateJob(ctx, designer)

	soon := time.Now().Add(24 * time.Hour)

	// Application 1: Jane -> Backend, two stages; phone screen has feedback submitted.
	app1 := &models.Application{JobID: backend.ID, CandidateID: jane.ID, Status: "active", CreatedBy: sched.ID}
	s.CreateApplication(ctx, app1)

	phone1 := &models.Stage{ApplicationID: app1.ID, Type: "phone_screen", FocusArea: "Recruiter Screen", ScheduledAt: soon, VideoLink: "https://meet.example.com/jane-phone", NotesForInterviewer: "Initial screen", Status: "pending"}
	s.CreateStage(ctx, phone1)
	s.AddStageInterviewer(ctx, phone1.ID, alice.ID)
	// Alice files feedback -> completes the phone screen (single interviewer).
	s.CreateFeedback(ctx, &models.Feedback{
		StageID: phone1.ID, InterviewerID: alice.ID, Recommendation: "hire",
		RecommendationReason: "Strong communicator, solid fundamentals.",
		FreeFormNotes:        "Good rapport, move forward to onsite.",
		CompetencyRatings: []models.CompetencyRating{
			{CompetencyID: communication.ID, RatingValue: "Advising"},
		},
	})

	coding1 := &models.Stage{ApplicationID: app1.ID, Type: "interview", FocusArea: "Coding", ScheduledAt: soon.Add(2 * time.Hour), VideoLink: "https://meet.example.com/jane-coding", NotesForInterviewer: "Data structures & algorithms", Status: "pending"}
	s.CreateStage(ctx, coding1)
	s.AddStageInterviewer(ctx, coding1.ID, bob.ID)

	// Application 2: Mike -> Backend, one interview stage with TWO interviewers (panel-ish, both must file).
	app2 := &models.Application{JobID: backend.ID, CandidateID: mike.ID, Status: "active", CreatedBy: sched.ID}
	s.CreateApplication(ctx, app2)
	sysd := &models.Stage{ApplicationID: app2.ID, Type: "interview", FocusArea: "System Design", ScheduledAt: soon.Add(3 * time.Hour), VideoLink: "https://meet.example.com/mike-design", NotesForInterviewer: "Distributed systems", Status: "pending"}
	s.CreateStage(ctx, sysd)
	s.AddStageInterviewer(ctx, sysd.ID, carol.ID)
	s.AddStageInterviewer(ctx, sysd.ID, dave.ID)

	// Application 3: Jane also applies to Designer (candidate on multiple jobs).
	app3 := &models.Application{JobID: designer.ID, CandidateID: jane.ID, Status: "active", CreatedBy: sched.ID}
	s.CreateApplication(ctx, app3)
	phone3 := &models.Stage{ApplicationID: app3.ID, Type: "phone_screen", FocusArea: "Portfolio Review", ScheduledAt: soon.Add(4 * time.Hour), VideoLink: "https://meet.example.com/jane-portfolio", NotesForInterviewer: "Walk through portfolio", Status: "pending"}
	s.CreateStage(ctx, phone3)
	s.AddStageInterviewer(ctx, phone3.ID, carol.ID)

	// A few notifications
	s.CreateNotification(ctx, &models.Notification{UserID: bob.ID, Message: "You've been assigned an interview", Link: fmt.Sprintf("/interviews/%d", coding1.ID)})
	s.CreateNotification(ctx, &models.Notification{UserID: carol.ID, Message: "You've been assigned an interview", Link: fmt.Sprintf("/interviews/%d", sysd.ID)})
	s.CreateNotification(ctx, &models.Notification{UserID: sched.ID, Message: "Feedback submitted for a phone screen", Link: fmt.Sprintf("/applications/%d", app1.ID)})

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
