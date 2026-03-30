package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/longbridge/openapi-go/content"
	"github.com/spf13/cobra"
)

func contentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "content",
		Short: "Content operations (topics, news)",
		Long:  `Query discussion topics and news for securities.`,
	}

	cmd.AddCommand(topicsCmd())
	cmd.AddCommand(newsCmd())

	return cmd
}

func topicsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "topics [symbol]",
		Short: "Get discussion topics for a symbol",
		Long: `Get discussion topics for a security symbol.

Examples:
  longbridge-fs content topics AAPL.US
  longbridge-fs content topics 700.HK --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTopics(args[0])
		},
	}

	return cmd
}

func newsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "news [symbol]",
		Short: "Get news for a symbol",
		Long: `Get news articles for a security symbol.

Examples:
  longbridge-fs content news TSLA.US
  longbridge-fs content news 9988.HK --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNews(args[0])
		},
	}

	return cmd
}

func runTopics(symbol string) error {
	ctx := context.Background()

	// Load credentials and create content context
	cc, err := createContentContext()
	if err != nil {
		return fmt.Errorf("failed to initialize content context: %w", err)
	}

	// Fetch topics
	topics, err := cc.Topics(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to fetch topics: %w", err)
	}

	// Output based on format
	switch outputFormat {
	case "json":
		return outputJSON(topics)
	default:
		return outputTopicsTable(topics)
	}
}

func runNews(symbol string) error {
	ctx := context.Background()

	// Load credentials and create content context
	cc, err := createContentContext()
	if err != nil {
		return fmt.Errorf("failed to initialize content context: %w", err)
	}

	// Fetch news
	news, err := cc.News(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to fetch news: %w", err)
	}

	// Output based on format
	switch outputFormat {
	case "json":
		return outputJSON(news)
	default:
		return outputNewsTable(news)
	}
}

func outputTopicsTable(topics []*content.TopicItem) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "| ID\t| Title\t| Published At\t| Comments\t| Likes\t| Shares\t|")
	fmt.Fprintln(w, "|----\t|-------\t|-------------\t|---------\t|------\t|-------\t|")

	for _, t := range topics {
		fmt.Fprintf(w, "| %s\t| %s\t| %s\t| %d\t| %d\t| %d\t|\n",
			truncate(t.Id, 12),
			truncate(t.Title, 50),
			t.PublishedAt.Format("2006-01-02 15:04"),
			t.CommentsCount,
			t.LikesCount,
			t.SharesCount,
		)
	}

	return w.Flush()
}

func outputNewsTable(news []*content.NewsItem) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "| ID\t| Title\t| Published At\t| Comments\t| Likes\t| Shares\t|")
	fmt.Fprintln(w, "|----\t|-------\t|-------------\t|---------\t|------\t|-------\t|")

	for _, n := range news {
		fmt.Fprintf(w, "| %s\t| %s\t| %s\t| %d\t| %d\t| %d\t|\n",
			truncate(n.Id, 12),
			truncate(n.Title, 50),
			n.PublishedAt.Format("2006-01-02 15:04"),
			n.CommentsCount,
			n.LikesCount,
			n.SharesCount,
		)
	}

	return w.Flush()
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
