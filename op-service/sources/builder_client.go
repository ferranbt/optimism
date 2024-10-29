package sources

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var _ driver.L2Chain = &BuilderClient{}

type BuilderClient struct {
	log      log.Logger
	fallback driver.L2Chain
	builder  *EngineClient
}

func NewBuilderClient(log log.Logger, fallback driver.L2Chain, builder *EngineClient) *BuilderClient {
	return &BuilderClient{
		log:      log,
		fallback: fallback,
		builder:  builder,
	}
}

func (b *BuilderClient) GetPayload(ctx context.Context, payloadInfo eth.PayloadInfo) (*eth.ExecutionPayloadEnvelope, error) {
	fPayload, err := b.fallback.GetPayload(ctx, payloadInfo)
	if err != nil {
		return nil, err
	}

	bPayload, err := b.builder.GetPayload(ctx, payloadInfo)
	if err != nil {
		log.Error("builder get payload failed", "err", err)
		return fPayload, nil
	}

	log.Info("return payload from builder", "payloadID", payloadInfo.ID)
	return bPayload, nil
}

func (b *BuilderClient) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	if attr == nil {
		// No block building
		return b.fallback.ForkchoiceUpdate(ctx, state, attr)
	}

	res, err := b.fallback.ForkchoiceUpdate(ctx, state, attr)
	if err != nil {
		return nil, err
	}
	if res.PayloadStatus.Status != eth.ExecutionValid {
		// return if the fallback node failed
		return res, nil
	}

	builderMultiplex := func() error {
		bRes, err := b.builder.ForkchoiceUpdate(ctx, state, attr)
		if err != nil {
			return err
		}
		if bRes.PayloadStatus.Status != eth.ExecutionValid {
			// the builder might not be synced yet
			return nil
		}
		if !bytes.Equal((*bRes.PayloadID)[:], (*res.PayloadID)[:]) {
			return fmt.Errorf("builder returned different payloadID: %s", bRes.PayloadID)
		}
		return nil
	}
	if err := builderMultiplex(); err != nil {
		// we should not fail the process if the builder failed
		b.log.Error("builder forkchoice update failed", "err", err)
	}

	return res, nil
}

func (b *BuilderClient) NewPayload(ctx context.Context, payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash) (*eth.PayloadStatusV1, error) {
	return b.fallback.NewPayload(ctx, payload, parentBeaconBlockRoot)
}

func (b *BuilderClient) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error) {
	return b.fallback.PayloadByHash(ctx, hash)
}

func (b *BuilderClient) PayloadByNumber(ctx context.Context, num uint64) (*eth.ExecutionPayloadEnvelope, error) {
	return b.fallback.PayloadByNumber(ctx, num)
}

func (b *BuilderClient) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	return b.fallback.L2BlockRefByLabel(ctx, label)
}

func (b *BuilderClient) L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error) {
	return b.fallback.L2BlockRefByHash(ctx, l2Hash)
}

func (b *BuilderClient) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	return b.fallback.L2BlockRefByNumber(ctx, num)
}

func (b *BuilderClient) SystemConfigByL2Hash(ctx context.Context, hash common.Hash) (eth.SystemConfig, error) {
	return b.fallback.SystemConfigByL2Hash(ctx, hash)
}
