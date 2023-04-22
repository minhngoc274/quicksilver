package initialization

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ingenuity-build/quicksilver/test/e2e/util"
)

func InitChain(id, dataDir string, nodeConfigs []*NodeConfig, votingPeriod time.Duration, forkHeight int) (*Chain, error) {
	chain := newInternal(id, dataDir)
	for _, nodeConfig := range nodeConfigs {
		newNode, err := newNode(chain, nodeConfig)
		if err != nil {
			return nil, err
		}
		chain.nodes = append(chain.nodes, newNode)
	}

	if err := initGenesis(chain, votingPeriod, forkHeight); err != nil {
		return nil, err
	}

	peers := make([]string, len(chain.nodes))
	for i, peer := range chain.nodes {
		peerID := fmt.Sprintf("%s@%s:26656", peer.getNodeKey().ID(), peer.moniker)
		peer.peerID = peerID
		peers[i] = peerID
	}

	for _, node := range chain.nodes {
		if node.isValidator {
			if err := node.initNodeConfigs(peers); err != nil {
				return nil, err
			}
		}
	}
	return chain.export()
}

func InitSingleNode(chainID, dataDir, existingGenesisDir string, nodeConfig *NodeConfig, trustHeight int64, trustHash string, stateSyncRPCServers, persistentPeers []string) (*Node, error) {
	if nodeConfig.IsValidator {
		return nil, errors.New("creating individual validator nodes after starting up chain is not currently supported")
	}

	chain := newInternal(chainID, dataDir)

	newNode, err := newNode(chain, nodeConfig)
	if err != nil {
		return nil, err
	}

	_, err = util.CopyFile(
		existingGenesisDir,
		filepath.Join(newNode.configDir(), "config", "genesis.json"),
	)
	if err != nil {
		return nil, err
	}

	if err := newNode.initNodeConfigs(persistentPeers); err != nil {
		return nil, err
	}

	if err := newNode.initStateSyncConfig(trustHeight, trustHash, stateSyncRPCServers); err != nil {
		return nil, err
	}

	return newNode.export()
}