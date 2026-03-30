package cmd

import (
	"product-service/internal/app"

	"github.com/spf13/cobra"
)

var workerUpdateStockCmd = &cobra.Command{
	Use:   "worker-update-stock",
	Short: "Jalankan consumer RabbitMQ untuk mengurangi stok produk (queue PRODUCT_UPDATE_STOCK_NAME)",
	Run: func(cmd *cobra.Command, args []string) {
		app.RunWorkerUpdateStock()
	},
}

func init() {
	rootCmd.AddCommand(workerUpdateStockCmd)
}
