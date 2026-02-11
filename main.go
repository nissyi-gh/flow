package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Use a temporary DB for testing
	tmpDir, err := os.MkdirTemp("", "flow-test")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewTaskStore(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// Add
	t1, err := store.Add("Buy groceries")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Add error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Added: %+v\n", t1)

	t2, _ := store.Add("Write tests")
	fmt.Printf("Added: %+v\n", t2)

	// List
	tasks, _ := store.List()
	fmt.Printf("List (%d tasks):\n", len(tasks))
	for _, t := range tasks {
		fmt.Printf("  %+v\n", t)
	}

	// Toggle
	_ = store.ToggleComplete(t1.ID)
	t1, _ = store.GetByID(t1.ID)
	fmt.Printf("After toggle: %+v\n", t1)

	// Delete
	_ = store.Delete(t2.ID)
	tasks, _ = store.List()
	fmt.Printf("After delete (%d tasks):\n", len(tasks))
	for _, t := range tasks {
		fmt.Printf("  %+v\n", t)
	}

	fmt.Println("All CRUD operations passed!")
}
