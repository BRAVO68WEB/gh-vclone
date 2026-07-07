package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/cli/go-gh/v2"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

type SearchOptions struct {
	Query string
}

type Repository struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func rootCmd() *cobra.Command {
	opts := &SearchOptions{}
	cmd := &cobra.Command{
		Use:   "gh vclone",
		Short: "gh vclone is a command line tool for GitHub repository clone",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("missing query")
			}
			opts.Query = args[0]

			return runSearch(opts)
		},
	}

	return cmd
}

func prepareQuery(opts *SearchOptions) string {
	return fmt.Sprintf("org:%s", opts.Query)
}

func pagingRepositories(repos []Repository, pageSize, page int) []Repository {
	if pageSize <= 0 {
		pageSize = 5
	}

	if len(repos) == 0 {
		return nil
	}

	start := page * pageSize
	if start >= len(repos) {
		return nil
	}

	end := start + pageSize
	if end > len(repos) {
		end = len(repos)
	}

	return repos[start:end]
}

func renderRepo(repoList []Repository, uname string) {
	currentPage := 0
	pageSize := 10
	totalPages := len(repoList) / pageSize
	if len(repoList)%pageSize != 0 {
		totalPages++
	}
	app := tview.NewApplication()

	list := tview.NewList()

	// Add a footer to the list
	list.SetBorder(true).SetTitle("Press Enter to select a repository")

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(list, 0, 1, true)

	render := func(page int) {
		list.Clear()
		Repos := pagingRepositories(repoList, pageSize, page)
		for _, repo := range Repos {
			list.AddItem(repo.Name, repo.Description, 0, nil)
		}
		layout.SetBorder(true).SetTitle(fmt.Sprintf("Page %d/%d", page+1, totalPages))
	}
	render(currentPage)
	userSelected := false
	userSelectedRepo := ""

	// Wait for user to select a repository and close the app and return the selected repository
	list.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		app.Stop()
		userSelected = true
		userSelectedRepo = mainText
	})
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			if currentPage > 0 {
				currentPage--
				render(currentPage)
			}
			return nil
		case tcell.KeyRight:
			if currentPage < totalPages-1 {
				currentPage++
				render(currentPage)
			}
			return nil
		}
		return event
	})

	if err := app.SetRoot(layout, true).Run(); err != nil {
		panic(err)
	}

	if userSelected {
		fmt.Printf("📚 Repository Selected: %s\n", userSelectedRepo)

		gh.Exec("repo", "clone", uname+"/"+userSelectedRepo)

		fmt.Printf("📂 Repository %s cloned\n", userSelectedRepo)

		fmt.Printf("☕ Happy coding!\n")
	}

	// If the user didn't select a repository, exit the program
	if !userSelected {
		fmt.Printf("🤷‍♀️ no repository selected\n")
	}
}

func runSearch(opts *SearchOptions) error {
	args := []string{"api", "users/" + opts.Query, "--jq", ".type"}

	accType, _, err := gh.Exec(args...)

	if err != nil {
		return err
	}

	accTypeString := accType.String()

	if accTypeString == "Organization\n" {
		fmt.Printf("🏛 Organization: %s\n\n", opts.Query)

		args := []string{"api", "orgs/" + opts.Query + "/repos", "--paginate"}

		repos, _, err := gh.Exec(args...)

		if err != nil {
			return err
		}

		repoList := []Repository{}

		err = json.Unmarshal(repos.Bytes(), &repoList)

		if err != nil {
			println("🐛 either the does not have any repositories or you dont have access to clone them")
			return err
		}

		renderRepo(repoList, opts.Query)

	} else if accTypeString == "User\n" {
		fmt.Printf("🧛‍♂️ User: %s\n", opts.Query)

		args := []string{"api", "users/" + opts.Query + "/repos", "--paginate"}

		repos, _, err := gh.Exec(args...)

		if err != nil {
			return err
		}

		repoList := []Repository{}

		err = json.Unmarshal(repos.Bytes(), &repoList)

		if err != nil {
			return err
		}

		renderRepo(repoList, opts.Query)
	} else {
		fmt.Printf("🤷‍♀️ %s is not a valid user or organization", opts.Query)
	}

	return nil
}

func main() {
	cmd := rootCmd()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
