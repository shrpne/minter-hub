package minter

import (
	c "context"
	"encoding/json"
	"fmt"
	oracleTypes "github.com/MinterTeam/mhub/chain/x/oracle/types"
	"github.com/MinterTeam/minter-go-sdk/v2/api/http_client"
	"github.com/MinterTeam/minter-go-sdk/v2/api/http_client/models"
	"github.com/MinterTeam/minter-go-sdk/v2/transaction"
	"github.com/MinterTeam/minter-hub-connector/command"
	"github.com/MinterTeam/minter-hub-connector/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"math"
	"sort"
	"strconv"
	"time"
)

func GetLatestMinterBlock(client *http_client.Client) uint64 {
	status, err := client.Status()
	if err != nil {
		println(err.Error())
		time.Sleep(1 * time.Second)
		return GetLatestMinterBlock(client)
	}

	return status.LatestBlockHeight
}

func GetLatestMinterBlockAndNonce(ctx context.Context, currentNonce uint64) context.Context {
	println("Current nonce @ hub", currentNonce)

	latestBlock := GetLatestMinterBlock(ctx.MinterClient)

	oracleClient := oracleTypes.NewQueryClient(ctx.CosmosConn)
	coinList, err := oracleClient.Coins(c.Background(), &oracleTypes.QueryCoinsRequest{})
	if err != nil {
		panic(err)
	}

	const blocksPerBatch = 100
	for i := uint64(0); i <= uint64(math.Ceil(float64(latestBlock-ctx.LastCheckedMinterBlock)/blocksPerBatch)); i++ {
		from := ctx.LastCheckedMinterBlock + 1 + i*blocksPerBatch
		to := ctx.LastCheckedMinterBlock + (i+1)*blocksPerBatch

		if to > latestBlock {
			to = latestBlock
		}

		blocks, err := ctx.MinterClient.Blocks(from, to, false)
		if err != nil {
			println("ERROR: ", err.Error())
			time.Sleep(time.Second)
			i--
			continue
		}

		sort.Slice(blocks.Blocks, func(i, j int) bool {
			return blocks.Blocks[i].Height < blocks.Blocks[j].Height
		})

		for _, block := range blocks.Blocks {
			fmt.Printf("\r%d of %d", block.Height, latestBlock)
			for _, tx := range block.Transactions {
				if tx.Type == uint64(transaction.TypeSend) {
					data, _ := tx.Data.UnmarshalNew()
					sendData := data.(*models.SendData)
					cmd := command.Command{}
					json.Unmarshal(tx.Payload, &cmd)

					value, _ := sdk.NewIntFromString(sendData.Value)
					if sendData.To == ctx.MinterMultisigAddr && cmd.Validate(value) == nil {
						for _, c := range coinList.GetCoins() {
							if sendData.Coin.ID == c.MinterId {
								if currentNonce < ctx.LastEventNonce {
									ctx.LastCheckedMinterBlock = block.Height - 1
									return ctx
								}

								ctx.LastEventNonce++
							}
						}
					}
				}

				if tx.Type == uint64(transaction.TypeMultisend) && tx.From == ctx.MinterMultisigAddr {
					if currentNonce < ctx.LastEventNonce {
						ctx.LastCheckedMinterBlock = block.Height - 1
						return ctx
					}

					ctx.LastEventNonce++
					ctx.LastBatchNonce++
				}

				if tx.Type == uint64(transaction.TypeEditMultisig) && tx.From == ctx.MinterMultisigAddr {
					nonce, err := strconv.Atoi(string(tx.Payload))
					if err != nil {
						println("ERROR:", err.Error())
					} else {
						if currentNonce < ctx.LastEventNonce {
							ctx.LastCheckedMinterBlock = block.Height - 1
							return ctx
						}

						ctx.LastValsetNonce = uint64(nonce)
						ctx.LastEventNonce++
					}
				}
			}

			ctx.LastCheckedMinterBlock = block.Height
		}
	}

	println()

	return ctx
}
