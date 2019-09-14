'use strict';

var hfc = require('fabric-client')
var path = require('path')
var sdkUtils = require('fabric-client/lib/utils') 
var fs = require('fs'); 
var util = require('util'); 
var express = require('express')
var bodyParser = require('body-parser')

var options = { 
    user_id: 'Admin@org1.zjucst.com', 
    msp_id:'Org1MSP', 
    channel_id: 'assetschannel', 
    chaincode_id: 'assets', 
    network_url: 'grpc://localhost:27051',
    peer_url: 'grpc://localhost:27051',
    orderer_url: 'grpc://localhost:7050',
    privateKeyFolder:'/Users/galeno/go/src/github.com/hyperledger/fabric-samples/food/crypto-config/peerOrganizations/org1.zjucst.com/users/Admin@org1.zjucst.com/msp/keystore', 
    signedCert:'/Users/galeno/go/src/github.com/hyperledger/fabric-samples/food/crypto-config/peerOrganizations/org1.zjucst.com/users/Admin@org1.zjucst.com/msp/signcerts/Admin@org1.zjucst.com-cert.pem', 
    peer_tls_cacerts:'/Users/galeno/go/src/github.com/hyperledger/fabric-samples/food/crypto-config/peerOrganizations/org1.zjucst.com/peers/peer0.org1.zjucst.com/tls/ca.crt', 
    orderer_tls_cacerts:'/Users/galeno/go/src/github.com/hyperledger/fabric-samples/food/crypto-config/ordererOrganizations/zjucst.com/orderers/orderer.zjucst.com/tls/ca.crt', 
    tls_cacerts:'/Users/galeno/go/src/github.com/hyperledger/fabric-samples/food/crypto-config/peerOrganizations/org1.zjucst.com/peers/peer0.org1.zjucst.com/tls/ca.crt', 
    server_hostname: "peer0.org1.zjucst.com" 
};

var channel = {}; 
var client = null; 
var targets = [];
var tx_id = null;
const getKeyFilesInDir = (dir) => { 
    //该函数用于找到keystore目录下的私钥文件的路径 
    var files = fs.readdirSync(dir) 
    var keyFiles = [] 
    files.forEach((file_name) => { 
        let filePath = path.join(dir, file_name) 
        if (file_name.endsWith('_sk')) { 
            keyFiles.push(filePath) 
        } 
    }) 
    return keyFiles 
} 

async function query(fcn,args){
    console.log("Load privateKey and signedCert"); 
    client = new hfc(); 
    var createUserOpt = { 
        username: options.user_id, 
        mspid: options.msp_id, 
        cryptoContent: { privateKey: getKeyFilesInDir(options.privateKeyFolder)[0], 
        signedCert: options.signedCert } 
    } 
    const store = await sdkUtils.newKeyValueStore({
        path: "/tmp/fabric-client-stateStore/"
    });
    client.setStateStore(store);
    let user = await client.createUser(createUserOpt);
    channel = client.newChannel(options.channel_id); 
    let data = fs.readFileSync(options.tls_cacerts); 
    let peer = client.newPeer(options.network_url, 
        { 
            pem: Buffer.from(data).toString(), 
            'ssl-target-name-override': options.server_hostname 
        } 
    ); 
    peer.setName("peer0");  
    channel.addPeer(peer);
    console.log("Start query"); 
    let transaction_id = await client.newTransactionID(); 
    console.log("Assigning transaction_id: ", transaction_id._transaction_id); 
    const request = { 
        chaincodeId: options.chaincode_id, 
        txId: transaction_id, 
        fcn: fcn, 
        args: args 
    };
    let query_responses = await channel.queryByChaincode(request);
    console.log("returned from query"); 
    if (!query_responses.length) { 
        console.log("No payloads were returned from query"); 
    } else { 
        console.log("Query result count = ", query_responses.length) 
    } 
    if (query_responses[0] instanceof Error) { 
        console.error("error from query = ", query_responses[0]); 
    } 
    console.log("Response is ", query_responses[0].toString());
    return query_responses[0].toString();
}

async function invoke(fcn,args) {
    console.log("Load privateKey and signedCert"); 
    client = new hfc(); 
    var createUserOpt = { 
        username: options.user_id, 
        mspid: options.msp_id, 
        cryptoContent: { privateKey: getKeyFilesInDir(options.privateKeyFolder)[0], 
        signedCert: options.signedCert } 
    } 
    const store = await sdkUtils.newKeyValueStore({
        path: "/tmp/fabric-client-stateStore/"
    });
    client.setStateStore(store);
    let user = await client.createUser(createUserOpt);
    channel = await client.newChannel(options.channel_id);
    let data = fs.readFileSync(options.peer_tls_cacerts); 
    let peer = client.newPeer(options.peer_url, 
        { 
            pem: Buffer.from(data).toString(), 
            'ssl-target-name-override': options.server_hostname 
        } 
    ); 
    channel.addPeer(peer); 
    let odata = fs.readFileSync(options.orderer_tls_cacerts); 
    let caroots = Buffer.from(odata).toString(); 
    var orderer = client.newOrderer(options.orderer_url, { 
        'pem': caroots, 
        'ssl-target-name-override': "orderer.zjucst.com" 
    }); 
    channel.addOrderer(orderer); 
    targets.push(peer);
    tx_id = client.newTransactionID(); 
    console.log("Assigning transaction_id: ", tx_id._transaction_id); 
    var request = { 
        targets: targets, 
        chaincodeId: options.chaincode_id, 
        fcn: fcn, 
        args: args, 
        chainId: options.channel_id, 
        txId: tx_id 
    }; 
    let results = await channel.sendTransactionProposal(request);
    var proposalResponses = results[0]; 
    var proposal = results[1]; 
    var header = results[2]; 
    let isGoodProposal = false; 
    if (proposalResponses && proposalResponses[0].response && 
        proposalResponses[0].response.status === 200) { 
        isGoodProposal = true; 
        console.log('transaction proposal was good'); 
    } else {
        console.log(results,isGoodProposal)
        console.error('transaction proposal was bad'); 
    }
    if (isGoodProposal){
        console.log(util.format(
            'Successfully sent Proposal and received ProposalResponse: Status - %s, message - "%s", metadata - "%s", endorsement signature: %s', 
            proposalResponses[0].response.status, proposalResponses[0].response.message, 
            proposalResponses[0].response.payload, proposalResponses[0].endorsement.signature
        ));
        var requests = { 
            proposalResponses: proposalResponses, 
            proposal: proposal, 
            header: header 
        }; 
        var transactionID = tx_id.getTransactionID(); 
        var eventPromises = []; 
        let eh = await channel.newChannelEventHub('localhost:27051');
        let data = fs.readFileSync(options.peer_tls_cacerts); 
        let grpcOpts = { 
            pem: Buffer.from(data).toString(), 
            'ssl-target-name-override': options.server_hostname 
        } 
        eh.connect();
        let txPromise = new Promise((resolve, reject) => { 
            let handle = setTimeout(() => { 
                eh.disconnect(); 
                reject(); 
            }, 30000); 
            eh.registerTxEvent(transactionID, (tx, code) => { 
                clearTimeout(handle); 
                eh.unregisterTxEvent(transactionID); 
                eh.disconnect();

                if (code !== 'VALID') { 
                    console.error( 
                        'The transaction was invalid, code = ' + code); 
                    reject(); 
                 } else { 
                    console.log( 
                        'The transaction has been committed on peer ' + 
                        eh.getPeerAddr());
                    resolve(); 
                } 
            }); 
        }); 
        eventPromises.push(txPromise); 
        var sendPromise = await channel.sendTransaction(requests);
        return Promise.all([sendPromise].concat(eventPromises)).then((result) => { 
            console.log(' event promise all complete');
            return result; 
        }).catch((err) => { 
            console.error( 
                'Failed to send transaction and get notifications within the timeout period.' 
            ); 
            return 'Failed to send transaction and get notifications within the timeout period.'; 
         }); 
    }else{
        console.error( 
            'Failed to send Proposal or receive valid response. Response null or status is not 200. exiting...' 
        ); 
        let result = 'failed'
        return result;
    }
}

var app = express()

app.use(express.static('public'));
app.use('/', express.static(__dirname + '/public'));
app.use(bodyParser.urlencoded({ extended: false }))
app.use(bodyParser.json())


app.get('/users',function (req,res) {
    let fcn = 'queryUser'
    let args = []
    args.push(req.query.id)
    //console.log(fcn,args)
    query(fcn,args).then((queryres) =>{
        res.send(queryres);
    })
})

app.get('/ingredients/get',function (req,res){
    let fcn = 'queryIngredient'
    let args =[]
    args.push(req.query.id)
    query(fcn,args).then((queryres) => {
        res.send(queryres)
    })
})

app.get('/users/:id',function (req,res) {
    let fcn = 'queryUser'
    let args = []
    args.push(req.params.id)
    //console.log(req)
    query(fcn,args).then((queryres) =>{
        res.send(queryres);
    })
})

app.get('/ingredients/get/:id',function (req,res){
    let fcn = 'queryIngredient'
    let args =[]
    args.push(req.params.id)
    query(fcn,args).then((queryres) => {
        res.send(queryres)
    })
})

app.get('/ingredients/exchange/history',function(req,res){
    let fcn = 'queryIngredientHistory'
    let args = []
    let obj = req.query.id
    let type = req.query.type
    args.push(obj)
    if (type!=undefined){
        args.push[type]
    }
    query(fcn,args).then((queryres) => {
        res.send(queryres)
    })
})

app.post('/users',function(req,res){
    let fcn = 'userRegister'
    let args = []
    //console.log(req)
    let id = req.body.id
    let name = req.body.name
    if(name != undefined) args.push(name)
    if(id !=undefined) args.push(id)
    console.log(fcn,args)
    invoke(fcn,args).then((result) => {
        res.send(result[0].status)
    })
})

app.post('/ingredients/enroll',function(req,res){
    let fcn = 'ingredientEnroll'
    let args = []
    let ingredientId=req.body.ingredientid
    let ingredientName=req.body.ingredientname
    let metadata=req.body.metadata
    let ownerId=req.body.ownerid
    if(ingredientId!=undefined) args.push(ingredientId)
    if(ingredientName!=undefined) args.push(ingredientName)
    if(metadata!=undefined) args.push(metadata)
    if(ownerId!=undefined) args.push(ownerId)
    console.log(fcn,args)
    invoke(fcn,args).then((result) => {
        res.send(result[0].status)
    })
})

app.post('/ingredients/exchange',function(req,res){
    let fcn = 'ingredientExchange'
    let args = []
    let OriginOwnerId = req.body.origin
    let IngredientId = req.body.id
    let CurrentOwnerId = req.body.current
    if(OriginOwnerId!=undefined) args.push(OriginOwnerId)
    if(IngredientId!=undefined) args.push(IngredientId)
    if(CurrentOwnerId!=undefined) args.push(CurrentOwnerId)
    console.log(fcn,args)
    invoke(fcn,args).then((result) => {
        res.send(result[0].status)
    })
})

app.get('/deleteusers',function(req,res){
    let fcn = 'userDestroy'
    let args = []
    args.push(req.query.id)
    invoke(fcn,args).then((result) => {
        res.send(result)
    })
})

app.delete('/deleteusers',function(req,res){
    let fcn = 'userDestroy'
    let args = []
    args.push(req.query.id)
    invoke(fcn,args).then((result) => {
        res.send(result)
    })
})

var server=app.listen(8080,function(){
    console.log("程序运行在8080端口")
})
