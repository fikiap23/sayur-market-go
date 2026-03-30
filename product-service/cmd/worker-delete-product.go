package cmd

import (
	"product-service/internal/app"

	"github.com/spf13/cobra"
)

var workerDeleteCmd = &cobra.Command{
	Use:   "worker-delete-product",
	Short: "Jalankan consumer RabbitMQ untuk menghapus produk dari Elasticsearch (queue PRODUCT_DELETE)",
	Run: func(cmd *cobra.Command, args []string) {
		app.RunWorkerESDelete()
	},
}

func init() {
	rootCmd.AddCommand(workerDeleteCmd)
}
