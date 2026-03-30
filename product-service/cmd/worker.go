package cmd

import (
	"product-service/internal/app"

	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Jalankan consumer RabbitMQ untuk mengindeks produk ke Elasticsearch (queue PRODUCT_PUBLISH_NAME)",
	Run: func(cmd *cobra.Command, args []string) {
		app.RunWorkerESIndex()
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)
}
