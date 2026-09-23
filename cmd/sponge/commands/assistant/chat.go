package assistant

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Eric-Guo/sponge/pkg/utils"
)

// ChatCommand chat with AI Assistant
func ChatCommand() *cobra.Command {
	var (
		assistantType string

		apiKey string
		model  string

		// chatgpt specific
		roleDesc    string
		maxToken    int
		temperature float32
	)

	//nolint
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Chat with AI Assistant",
		Long:  "Chat with AI Assistant.",
		Example: color.HiBlackString(` # Running ChatGPT assistant, default model is gpt-4o.
  sponge assistant chat --type=chatgpt --api-key=your-api-key

  # Running DeepSeek assistant, default model is deepseek-chat.
  sponge assistant chat --type=deepseek --api-key=your-api-key

  # Running Gemini assistant, default model is gemini-2.5-pro-exp.
  sponge assistant chat --type=gemini --api-key=your-api-key

  # Running assistant with custom model, e.g. deepseek with model deepseek-reasoner, 
  # chatgpt with model o1-mini, gemini with model gemini-2.0-flash.
  sponge assistant chat --type=deepseek --api-key=your-api-key --model=deepseek-reasoner`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			asst := &assistantParams{
				Type:          assistantType,
				apiKey:        apiKey,
				model:         model,
				enableContext: true,
				roleDesc:      roleDesc,
				maxToken:      maxToken,
				temperature:   temperature,
			}
			return chat(asst)
		},
	}

	cmd.Flags().StringVarP(&assistantType, "type", "t", "", "assistant type, e.g. chatgpt, deepseek")
	_ = cmd.MarkFlagRequired("type")
	cmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "assistant api key")
	_ = cmd.MarkFlagRequired("api-key")
	cmd.Flags().StringVarP(&model, "model", "m", "", "assistant model, corresponding assistant type.")
	cmd.Flags().StringVarP(&roleDesc, "role", "r", "", "role description, for example, you are a psychologist.")
	cmd.Flags().IntVarP(&maxToken, "max-token", "s", 0, "maximum number of tokens")
	cmd.Flags().Float32VarP(&temperature, "temperature", "e", 0, "temperature of the model")

	return cmd
}

func chat(asst *assistantParams) error {
	client, err := asst.newClient()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter questions and talk to the assistant, enter 'q' or 'quit' to exit, enter 'r' to refresh context.")
	for {
		fmt.Print(color.HiCyanString("Prompt: "))

		input, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("reading input error: %v", err)
		}
		input = strings.TrimSpace(input)
		fmt.Println()

		if input == "q" || input == "quit" || input == "exit" {
			fmt.Println("Exited assistant.")
			break
		}
		if input == "r" || input == "R" {
			client.RefreshContext()
			fmt.Println(color.HiBlackString("Start a new conversation.") + "\n\n")
			continue
		}

		answer := client.SendStream(context.Background(), input)
		p := utils.NewWaitPrinter(time.Millisecond * 200)
		p.LoopPrint(assistantTypeMap[asst.Type] + ": ")
		isFirst := true
		for content := range answer.Content {
			if isFirst {
				isFirst = false
				p.StopPrint(content)
			} else {
				fmt.Print(content)
			}
		}
		if answer.Err != nil {
			return fmt.Errorf("error : %v", answer.Err)
		}
		fmt.Printf("\n\n\n")
	}
	return nil
}
