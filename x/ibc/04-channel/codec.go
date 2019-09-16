package channel

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

var msgCdc *codec.Codec

func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterInterface((*Packet)(nil), nil)
}

func SetMsgCodec(cdc *codec.Codec) {
	// TODO
	/*
		if msgCdc != nil && msgCdc != cdc {
			panic("MsgCdc set more than once")
		}
	*/
	msgCdc = cdc
}