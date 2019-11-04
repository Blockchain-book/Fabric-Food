## 删除所有容器
docker rm $(docker ps -aq)

## 设置工作路径
export FABRICPATH=$GOPATH/src/github.com/hyperledger/fabric-samples
export FABRIC_CFG_PATH=$FABRICPATH/trash


## 环境清理
rm -fr config/*
rm -fr crypto-config/*

## 生成证书文件
..$FABRICPATH/bin/cryptogen generate --config=./crypto-config.yaml

## 生成创世区块
..$FABRICPATH/bin/configtxgen -profile OneOrgOrdererGenesis -outputBlock ./config/genesis.block

## 生成通道的创世交易
..$FABRICPATH/bin/configtxgen -profile TwoOrgChannel -outputCreateChannelTx ./config/mychannel.tx -channelID mychannel
..$FABRICPATH/bin/configtxgen -profile TwoOrgChannel -outputCreateChannelTx ./config/assetschannel.tx -channelID assetschannel

## 生成组织关于通道的锚节点（主节点）交易
..$FABRICPATH/bin/configtxgen -profile TwoOrgChannel -outputAnchorPeersUpdate ./config/Org0MSPanchors.tx -channelID mychannel -asOrg Org0MSP
..$FABRICPATH/bin/configtxgen -profile TwoOrgChannel -outputAnchorPeersUpdate ./config/Org1MSPanchors.tx -channelID mychannel -asOrg Org1MSP

## 启动网络
docker-compose -f docker-compose.yaml up -d

## 进入CLI容器
docker exec -it cli bash

## 创建通道
peer channel create -o orderer.zjucst.com:7050 -c mychannel -f /etc/hyperledger/config/mychannel.tx
peer channel create -o orderer.zjucst.com:7050 -c assetschannel -f /etc/hyperledger/config/assetschannel.tx

## 加入通道
peer channel join -b mychannel.block
peer channel join -b assetschannel.block

## 设置主节点
peer channel update -o orderer.zjucst.com:7050 -c mychannel -f /etc/hyperledger/config/Org1MSPanchors.tx

## 链码安装
peer chaincode install -n assets -v 1.0 -l golang -p github.com/food

## 链码实例化
peer chaincode instantiate -o orderer.zjucst.com:7050 -C assetschannel -n assets -l golang -v 1.0 -c '{"Args":["init"]}'

# 链码交互操作或者客户端操作

## 链码交互
peer chaincode invoke -C assetschannel -n assets -c '{"Args":["userRegister", "user1", "user1"]}'
peer chaincode invoke -C assetschannel -n assets -c '{"Args":["ingredientEnroll", "assets1", "assets1", "metadata", "user1"]}'
peer chaincode invoke -C assetschannel -n assets -c '{"Args":["foodEnroll", "food1", "food1", "metadata", "user1"]}'
peer chaincode invoke -C assetschannel -n assets -c '{"Args":["userRegister", "user2", "user2"]}'
peer chaincode invoke -C assetschannel -n assets -c '{"Args":["ingredientExchange", "user1", "assets1", "user2"]}'
peer chaincode invoke -C assetschannel -n assets -c '{"Args":["userDestroy", "user1"]}'

## 链码升级
peer chaincode install -n assets -v 1.0.1 -l golang -p github.com/chaincode/assetsExchange
peer chaincode upgrade -C assetschannel -n assets -v 1.0.1 -c '{"Args":[""]}'

## 链码查询
peer chaincode query -C assetschannel -n assets -c '{"Args":["queryUser", "user1"]}'
peer chaincode query -C assetschannel -n assets -c '{"Args":["queryIngredient", "asset1"]}'
peer chaincode query -C assetschannel -n assets -c '{"Args":["queryUser", "user2"]}'
peer chaincode query -C assetschannel -n assets -c '{"Args":["queryIngredientHistory", "assets1"]}'
peer chaincode query -C assetschannel -n assets -c '{"Args":["queryIngredientHistory", "asset1", "all"]}'

## 命令行模式的背书策略

EXPR(E[,E...])
EXPR = OR AND
E = EXPR(E[,E...])
MSP.ROLE
MSP 组织名 org0MSP org1MSP
ROLE admin member

OR('org0MSP.member','org1MSP.admin')

## 在dev模式下运行链码
CORE_CHAINCODE_ID_NAME=assets:1.0.0 CORE_PEER_ADDRESS=0.0.0.0:27051 CORE_CHAINCODE_LOGGING_LEVEL=DEBUG go run -tags=nopkcs11 assetsExchange.go


## Open Google Browser
/usr/bin/google-chrome-stable