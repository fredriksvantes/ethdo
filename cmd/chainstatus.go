// Copyright © 2020 Weald Technology Trading
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
	spec "github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wealdtech/ethdo/util"
)

var chainStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Obtain status about a chain",
	Long: `Obtain status about a chain.  For example:

    ethdo chain status

In quiet mode this will return 0 if the chain status can be obtained, otherwise 1.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		eth2Client, err := util.ConnectToBeaconNode(ctx, viper.GetString("connection"), viper.GetDuration("timeout"), viper.GetBool("allow-insecure-connections"))
		errCheck(err, "Failed to connect to Ethereum 2 beacon node")

		specProvider, isProvider := eth2Client.(eth2client.SpecProvider)
		assert(isProvider, "beacon node does not provide spec; cannot report on chain status")
		config, err := specProvider.Spec(ctx)
		errCheck(err, "Failed to obtain beacon chain specification")

		finalityProvider, isProvider := eth2Client.(eth2client.FinalityProvider)
		assert(isProvider, "beacon node does not provide finality; cannot report on chain status")
		finality, err := finalityProvider.Finality(ctx, "head")
		errCheck(err, "Failed to obtain finality information")

		genesisProvider, isProvider := eth2Client.(eth2client.GenesisProvider)
		assert(isProvider, "beacon node does not provide genesis; cannot report on chain status")
		genesis, err := genesisProvider.Genesis(ctx)
		errCheck(err, "Failed to obtain genesis information")

		slotDuration := config["SECONDS_PER_SLOT"].(time.Duration)
		curSlot := timestampToSlot(genesis.GenesisTime, time.Now(), slotDuration)
		slotsPerEpoch := config["SLOTS_PER_EPOCH"].(uint64)
		curEpoch := spec.Epoch(uint64(curSlot) / slotsPerEpoch)
		fmt.Printf("Current epoch: %d\n", curEpoch)
		outputIf(verbose, fmt.Sprintf("Current slot: %d", curSlot))
		fmt.Printf("Justified epoch: %d\n", finality.Justified.Epoch)
		if verbose {
			distance := curEpoch - finality.Justified.Epoch
			fmt.Printf("Justified epoch distance: %d\n", distance)
		}
		fmt.Printf("Finalized epoch: %d\n", finality.Finalized.Epoch)
		if verbose {
			distance := curEpoch - finality.Finalized.Epoch
			fmt.Printf("Finalized epoch distance: %d\n", distance)
		}
		if verbose {
			fmt.Printf("Prior justified epoch: %d\n", finality.PreviousJustified.Epoch)
			distance := curEpoch - finality.PreviousJustified.Epoch
			fmt.Printf("Prior justified epoch distance: %d\n", distance)
		}

		if verbose {
			epochStartSlot := (uint64(curSlot) / slotsPerEpoch) * slotsPerEpoch
			fmt.Printf("Epoch slots: %d-%d\n", epochStartSlot, epochStartSlot+slotsPerEpoch-1)
			nextSlotTimestamp := slotToTimestamp(genesis.GenesisTime, curSlot+1, slotDuration)
			fmt.Printf("Time until next slot: %2.1fs\n", float64(time.Until(time.Unix(nextSlotTimestamp, 0)).Milliseconds())/1000)
			nextEpoch := epochToTimestamp(genesis.GenesisTime, spec.Slot(uint64(curSlot)/slotsPerEpoch+1), slotDuration, slotsPerEpoch)
			fmt.Printf("Slots until next epoch: %d\n", (uint64(curSlot)/slotsPerEpoch+1)*slotsPerEpoch-uint64(curSlot))
			fmt.Printf("Time until next epoch: %2.1fs\n", float64(time.Until(time.Unix(nextEpoch, 0)).Milliseconds())/1000)
		}

		os.Exit(_exitSuccess)
	},
}

func init() {
	chainCmd.AddCommand(chainStatusCmd)
	chainFlags(chainStatusCmd)
}
