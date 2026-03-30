package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/longbridge/openapi-go/content"
	"github.com/longbridge/openapi-go/http"
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
	cmd.AddCommand(topicDetailCmd())
	cmd.AddCommand(topicRepliesCmd())
	cmd.AddCommand(createTopicCmd())
	cmd.AddCommand(createReplyCmd())
	cmd.AddCommand(myTopicsCmd())

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

// Additional Community API commands

func topicDetailCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "topic-detail [topic_id]",
		Short: "Get detailed information about a topic",
		Long: `Get detailed information about a specific community topic.

Examples:
  longbridge-fs content topic-detail 6993508780031016960
  longbridge-fs content topic-detail 6993508780031016960 --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTopicDetail(args[0])
		},
	}

	return cmd
}

func topicRepliesCmd() *cobra.Command {
	var page, perPage int

	cmd := &cobra.Command{
		Use:   "topic-replies [topic_id]",
		Short: "List replies to a topic",
		Long: `List all replies to a community topic.

Examples:
  longbridge-fs content topic-replies 6993508780031016960
  longbridge-fs content topic-replies 6993508780031016960 --page 2 --per-page 50`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTopicReplies(args[0], page, perPage)
		},
	}

	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&perPage, "per-page", 30, "Results per page")

	return cmd
}

func createTopicCmd() *cobra.Command {
	var (
		title      string
		body       string
		topicType  string
		tickers    []string
		hashtags   []string
		license    int
	)

	cmd := &cobra.Command{
		Use:   "create-topic",
		Short: "Create a new community topic",
		Long: `Create a new community topic.

Examples:
  longbridge-fs content create-topic --title "My AAPL Analysis" --body "Apple reported..." --tickers AAPL.US
  longbridge-fs content create-topic --title "Market Update" --body "..." --topic-type article --license 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateTopic(title, body, topicType, tickers, hashtags, license)
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Topic title (required)")
	cmd.Flags().StringVar(&body, "body", "", "Topic body in Markdown (required)")
	cmd.Flags().StringVar(&topicType, "topic-type", "post", "Topic type (post or article)")
	cmd.Flags().StringSliceVar(&tickers, "tickers", []string{}, "Stock symbols (e.g., AAPL.US,TSLA.US)")
	cmd.Flags().StringSliceVar(&hashtags, "hashtags", []string{}, "Hashtags (max 5)")
	cmd.Flags().IntVar(&license, "license", 0, "License type (0=none, 1=original, 2=non-original)")

	cmd.MarkFlagRequired("title")
	cmd.MarkFlagRequired("body")

	return cmd
}

func createReplyCmd() *cobra.Command {
	var (
		body      string
		replyToID string
	)

	cmd := &cobra.Command{
		Use:   "create-reply [topic_id]",
		Short: "Create a reply to a topic",
		Long: `Create a reply to a community topic.

Examples:
  longbridge-fs content create-reply 6993508780031016960 --body "Great analysis!"
  longbridge-fs content create-reply 6993508780031016960 --body "I agree" --reply-to "7123456"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateReply(args[0], body, replyToID)
		},
	}

	cmd.Flags().StringVar(&body, "body", "", "Reply text (required)")
	cmd.Flags().StringVar(&replyToID, "reply-to", "0", "Parent reply ID (0 for top-level)")

	cmd.MarkFlagRequired("body")

	return cmd
}

func myTopicsCmd() *cobra.Command {
	var page, perPage int

	cmd := &cobra.Command{
		Use:   "my-topics",
		Short: "List my published topics",
		Long: `List topics published by the authenticated user.

Examples:
  longbridge-fs content my-topics
  longbridge-fs content my-topics --page 2 --per-page 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMyTopics(page, perPage)
		},
	}

	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&perPage, "per-page", 30, "Results per page")

	return cmd
}

// Implementation functions

type TopicDetail struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Body          string    `json:"body"`
	TopicType     string    `json:"topic_type"`
	Tickers       []string  `json:"tickers"`
	Hashtags      []string  `json:"hashtags"`
	CommentsCount int32     `json:"comments_count"`
	LikesCount    int32     `json:"likes_count"`
	SharesCount   int32     `json:"shares_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type TopicReply struct {
	ID          string    `json:"id"`
	TopicID     string    `json:"topic_id"`
	Body        string    `json:"body"`
	ReplyToID   string    `json:"reply_to_id"`
	LikesCount  int32     `json:"likes_count"`
	CreatedAt   time.Time `json:"created_at"`
	AuthorID    string    `json:"author_id"`
	AuthorName  string    `json:"author_name"`
}

type CreateTopicRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	TopicType string   `json:"topic_type,omitempty"`
	Tickers   []string `json:"tickers,omitempty"`
	Hashtags  []string `json:"hashtags,omitempty"`
	License   int      `json:"license,omitempty"`
}

type CreateReplyRequest struct {
	Body      string `json:"body"`
	ReplyToID string `json:"reply_to_id,omitempty"`
}

func runTopicDetail(topicID string) error {
	ctx := context.Background()

	cfg, err := createHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	httpClient, err := http.NewFromCfg(cfg)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	var result struct {
		Data TopicDetail `json:"data"`
	}

	err = httpClient.Get(ctx, fmt.Sprintf("/v1/content/topics/%s", topicID), url.Values{}, &result)
	if err != nil {
		return fmt.Errorf("failed to fetch topic detail: %w", err)
	}

	switch outputFormat {
	case "json":
		return outputJSON(result.Data)
	default:
		fmt.Printf("Topic ID: %s\n", result.Data.ID)
		fmt.Printf("Title: %s\n", result.Data.Title)
		fmt.Printf("Type: %s\n", result.Data.TopicType)
		fmt.Printf("Tickers: %v\n", result.Data.Tickers)
		fmt.Printf("Hashtags: %v\n", result.Data.Hashtags)
		fmt.Printf("Comments: %d\n", result.Data.CommentsCount)
		fmt.Printf("Likes: %d\n", result.Data.LikesCount)
		fmt.Printf("Shares: %d\n", result.Data.SharesCount)
		fmt.Printf("Created: %s\n", result.Data.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("\nBody:\n%s\n", result.Data.Body)
		return nil
	}
}

func runTopicReplies(topicID string, page, perPage int) error {
	ctx := context.Background()

	cfg, err := createHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	httpClient, err := http.NewFromCfg(cfg)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if perPage > 0 {
		params.Set("per_page", strconv.Itoa(perPage))
	}

	var result struct {
		Data struct {
			Items []TopicReply `json:"items"`
		} `json:"data"`
	}

	err = httpClient.Get(ctx, fmt.Sprintf("/v1/content/topics/%s/comments", topicID), params, &result)
	if err != nil {
		return fmt.Errorf("failed to fetch topic replies: %w", err)
	}

	switch outputFormat {
	case "json":
		return outputJSON(result.Data.Items)
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "| ID\t| Author\t| Body\t| Likes\t| Created At\t|")
		fmt.Fprintln(w, "|----\t|--------\t|------\t|-------\t|-----------\t|")

		for _, r := range result.Data.Items {
			fmt.Fprintf(w, "| %s\t| %s\t| %s\t| %d\t| %s\t|\n",
				truncate(r.ID, 12),
				truncate(r.AuthorName, 15),
				truncate(r.Body, 50),
				r.LikesCount,
				r.CreatedAt.Format("2006-01-02 15:04"),
			)
		}
		return w.Flush()
	}
}

func runCreateTopic(title, body, topicType string, tickers, hashtags []string, license int) error {
	ctx := context.Background()

	cfg, err := createHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	httpClient, err := http.NewFromCfg(cfg)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	req := CreateTopicRequest{
		Title:     title,
		Body:      body,
		TopicType: topicType,
		Tickers:   tickers,
		Hashtags:  hashtags,
		License:   license,
	}

	var result struct {
		Data struct {
			ID        string   `json:"id"`
			Title     string   `json:"title"`
			TopicType string   `json:"topic_type"`
			Tickers   []string `json:"tickers"`
			Hashtags  []string `json:"hashtags"`
			CreatedAt int64    `json:"created_at"`
		} `json:"data"`
	}

	err = httpClient.Post(ctx, "/v1/content/topics", req, &result)
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	switch outputFormat {
	case "json":
		return outputJSON(result.Data)
	default:
		fmt.Printf("✓ Topic created successfully\n")
		fmt.Printf("Topic ID: %s\n", result.Data.ID)
		fmt.Printf("Title: %s\n", result.Data.Title)
		fmt.Printf("Type: %s\n", result.Data.TopicType)
		return nil
	}
}

func runCreateReply(topicID, body, replyToID string) error {
	ctx := context.Background()

	cfg, err := createHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	httpClient, err := http.NewFromCfg(cfg)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	req := CreateReplyRequest{
		Body:      body,
		ReplyToID: replyToID,
	}

	var result struct {
		Data struct {
			ID          string `json:"id"`
			TopicID     string `json:"topic_id"`
			Body        string `json:"body"`
			ReplyToID   string `json:"reply_to_id"`
			CreatedAt   int64  `json:"created_at"`
		} `json:"data"`
	}

	err = httpClient.Post(ctx, fmt.Sprintf("/v1/content/topics/%s/comments", topicID), req, &result)
	if err != nil {
		return fmt.Errorf("failed to create reply: %w", err)
	}

	switch outputFormat {
	case "json":
		return outputJSON(result.Data)
	default:
		fmt.Printf("✓ Reply created successfully\n")
		fmt.Printf("Reply ID: %s\n", result.Data.ID)
		fmt.Printf("Topic ID: %s\n", result.Data.TopicID)
		return nil
	}
}

func runMyTopics(page, perPage int) error {
	ctx := context.Background()

	cfg, err := createHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	httpClient, err := http.NewFromCfg(cfg)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if perPage > 0 {
		params.Set("per_page", strconv.Itoa(perPage))
	}

	var result struct {
		Data struct {
			Items []TopicDetail `json:"items"`
		} `json:"data"`
	}

	err = httpClient.Get(ctx, "/v1/content/my/topics", params, &result)
	if err != nil {
		return fmt.Errorf("failed to fetch my topics: %w", err)
	}

	switch outputFormat {
	case "json":
		return outputJSON(result.Data.Items)
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "| ID\t| Title\t| Type\t| Comments\t| Likes\t| Created At\t|")
		fmt.Fprintln(w, "|----\t|-------\t|------\t|---------\t|------\t|-----------\t|")

		for _, t := range result.Data.Items {
			fmt.Fprintf(w, "| %s\t| %s\t| %s\t| %d\t| %d\t| %s\t|\n",
				truncate(t.ID, 12),
				truncate(t.Title, 50),
				t.TopicType,
				t.CommentsCount,
				t.LikesCount,
				t.CreatedAt.Format("2006-01-02 15:04"),
			)
		}
		return w.Flush()
	}
}
