package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type IngredientsExchangeCC struct{}

const (
	originOwner = "originPlaceholder"
)

// 用户
type User struct {
	Name        string   `json:"name"`
	Id          string   `json:"id"`
	Ingredients []string `json:"ingredients"`
	Foods       []string `json:"foods"`
}

// 食品
type Food struct {
	Name        string   `json:"name"`
	Id          string   `json:"id"`
	Metadata    string   `json:"metadata"`
	Ingredients []string `json:"ingredients"`
}

// 食材
type Ingredient struct {
	Name     string `json:"name"`
	Id       string `json:"id"`
	Metadata string `json:"metadata"`
}

// 食材流通
type IngredientHistory struct {
	IngredientId   string `json:"ingredient_id"`
	OriginOwnerId  string `json:"origin_owner_id"`
	CurrentOwnerId string `json:"current_owner_id"`
}

// 外卖流通
type FoodHistory struct {
	FoodId         string `json:"food_id"`
	OriginOwnerId  string `json:"origin_owner_id"`
	CurrentOwnerId string `json:"current_owner_id"`
}

func constructUserKey(userId string) string {
	return fmt.Sprintf("user_%s", userId)
}

func constructFOODKey(foodId string) string {
	return fmt.Sprintf("food_%s", foodId)
}

func constructIngredientKey(ingredientId string) string {
	return fmt.Sprintf("ingredient_%s", ingredientId)
}

func constructFoodKey(foodId string) string {
	return fmt.Sprintf("food_%s", foodId)
}

// 用户注册
func (c *IngredientsExchangeCC) userRegister(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//检查参数的个数
	if len(args) != 2 {
		return shim.Error("not enough args")
	}

	//验证参数的正确性
	name := args[0]
	id := args[1]
	if name == "" || id == "" {
		return shim.Error("invalid args")
	}

	//验证数据是否存在
	if userBytes, err := stub.GetState(constructUserKey(id)); err == nil && len(userBytes) != 0 {
		return shim.Error("user already exist")
	}

	//写入状态
	user := &User{
		Name:        name,
		Id:          id,
		Ingredients: make([]string, 0),
		Foods:       make([]string, 0),
	}

	// 序列化对象
	userBytes, err := json.Marshal(user)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error %s", err))
	}

	if err := stub.PutState(constructUserKey(id), userBytes); err != nil {
		return shim.Error(fmt.Sprintf("put user error %s", err))
	}

	// 成功返回
	return shim.Success(nil)
}

// 删除用户
func (c *IngredientsExchangeCC) userDestroy(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//检查参数的个数
	if len(args) != 1 {
		return shim.Error("not enough args")
	}

	//验证参数的正确性
	id := args[0]
	if id == "" {
		return shim.Error("invalid args")
	}

	//验证数据是否存在
	userBytes, err := stub.GetState(constructUserKey(id))
	if err != nil || len(userBytes) == 0 {
		return shim.Error("user not found")
	}

	//写入状态
	if err := stub.DelState(constructUserKey(id)); err != nil {
		return shim.Error(fmt.Sprintf("delete user error: %s", err))
	}

	// 删除用户名下的食材
	user := new(User)
	if err := json.Unmarshal(userBytes, user); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	for _, ingredientid := range user.Ingredients {
		if err := stub.DelState(constructIngredientKey(ingredientid)); err != nil {
			return shim.Error(fmt.Sprintf("delete ingredient error: %s", err))
		}
	}

	return shim.Success(nil)
}

// 食材登记
func (c *IngredientsExchangeCC) ingredientEnroll(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//检查参数的个数
	if len(args) != 4 {
		return shim.Error("not enough args")
	}

	//验证参数的正确性
	ingredientName := args[0]
	ingredientId := args[1]
	metadata := args[2]
	ownerId := args[3]
	if ingredientName == "" || ingredientId == "" || ownerId == "" {
		return shim.Error("invalid args")
	}

	//验证数据是否存在
	userBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(userBytes) == 0 {
		return shim.Error("user not found")
	}

	if ingredientBytes, err := stub.GetState(constructIngredientKey(ingredientId)); err == nil && len(ingredientBytes) != 0 {
		return shim.Error("ingredient already exist")
	}

	//写入状态
	ingredient := &Ingredient{
		Name:     ingredientName,
		Id:       ingredientId,
		Metadata: metadata,
	}
	ingredientBytes, err := json.Marshal(ingredient)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal ingredient error: %s", err))
	}
	if err := stub.PutState(constructIngredientKey(ingredientId), ingredientBytes); err != nil {
		return shim.Error(fmt.Sprintf("save ingredient error: %s", err))
	}

	user := new(User)
	// 反序列化用户
	if err := json.Unmarshal(userBytes, user); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	user.Ingredients = append(user.Ingredients, ingredientId)
	// 序列化用户
	userBytes, err = json.Marshal(user)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(user.Id), userBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 食材变更历史
	history := &IngredientHistory{
		IngredientId:   ingredientId,
		OriginOwnerId:  originOwner,
		CurrentOwnerId: ownerId,
	}
	historyBytes, err := json.Marshal(history)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal ingredient history error: %s", err))
	}

	historyKey, err := stub.CreateCompositeKey("history", []string{
		ingredientId,
		originOwner,
		ownerId,
	})
	if err != nil {
		return shim.Error(fmt.Sprintf("create key error: %s", err))
	}

	if err := stub.PutState(historyKey, historyBytes); err != nil {
		return shim.Error(fmt.Sprintf("save ingredient history error: %s", err))
	}

	return shim.Success(nil)
}

//食材登记
func (c *IngredientsExchangeCC) foodEnroll(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//检查参数的个数
	if len(args) != 4 {
		return shim.Error("not enough args")
	}

	//验证参数的正确性
	foodName := args[0]
	foodId := args[1]
	metadata := args[2]
	ownerId := args[3]
	if foodName == "" || foodId == "" || ownerId == "" {
		return shim.Error("invalid args")
	}

	//验证数据是否存在
	userBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(userBytes) == 0 {
		return shim.Error("user not found")
	}

	if foodBytes, err := stub.GetState(constructFOODKey(foodId)); err == nil && len(foodBytes) != 0 {
		return shim.Error("food already exist")
	}

	//写入状态
	food := &Food{
		Name:     foodName,
		Id:       foodId,
		Metadata: metadata,
	}
	foodBytes, err := json.Marshal(food)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal food error: %s", err))
	}
	if err := stub.PutState(constructFoodKey(foodId), foodBytes); err != nil {
		return shim.Error(fmt.Sprintf("save food error: %s", err))
	}

	user := new(User)
	// 反序列化用户
	if err := json.Unmarshal(userBytes, user); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	user.Foods = append(user.Foods, foodId)
	// 序列化用户
	userBytes, err = json.Marshal(user)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(user.Id), userBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	//外卖变更历史
	history := &FoodHistory{
		FoodId:         foodId,
		OriginOwnerId:  originOwner,
		CurrentOwnerId: ownerId,
	}
	historyBytes, err := json.Marshal(history)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal food history error: %s", err))
	}

	historyKey, err := stub.CreateCompositeKey("history", []string{
		foodId,
		originOwner,
		ownerId,
	})
	if err != nil {
		return shim.Error(fmt.Sprintf("create key error: %s", err))
	}

	if err := stub.PutState(historyKey, historyBytes); err != nil {
		return shim.Error(fmt.Sprintf("save food history error: %s", err))
	}

	return shim.Success(nil)
}

// 食材变更
func (c *IngredientsExchangeCC) ingredientExchange(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//检查参数的个数
	if len(args) != 3 {
		return shim.Error("not enough args")
	}

	//验证参数的正确性
	ownerId := args[0]
	ingredientId := args[1]
	currentOwnerId := args[2]
	if ownerId == "" || ingredientId == "" || currentOwnerId == "" {
		return shim.Error("invalid args")
	}

	//验证数据是否存在
	originOwnerBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(originOwnerBytes) == 0 {
		return shim.Error("user not found")
	}

	currentOwnerBytes, err := stub.GetState(constructUserKey(currentOwnerId))
	if err != nil || len(currentOwnerBytes) == 0 {
		return shim.Error("user not found")
	}

	assetBytes, err := stub.GetState(constructIngredientKey(ingredientId))
	if err != nil || len(assetBytes) == 0 {
		return shim.Error("asset not found")
	}

	// 校验原始拥有者确实拥有当前变更的食材
	originOwner := new(User)
	// 反序列化用户
	if err := json.Unmarshal(originOwnerBytes, originOwner); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	aidexist := false
	for _, aid := range originOwner.Ingredients {
		if aid == ingredientId {
			aidexist = true
			break
		}
	}
	if !aidexist {
		return shim.Error("ingredient owner not match")
	}

	//写入状态
	ingredientIds := make([]string, 0)
	for _, aid := range originOwner.Ingredients {
		if aid == ingredientId {
			continue
		}

		ingredientIds = append(ingredientIds, aid)
	}
	originOwner.Ingredients = ingredientIds

	originOwnerBytes, err = json.Marshal(originOwner)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(ownerId), originOwnerBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 当前拥有者插入食材id
	currentOwner := new(User)
	// 反序列化用户
	if err := json.Unmarshal(currentOwnerBytes, currentOwner); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	currentOwner.Ingredients = append(currentOwner.Ingredients, ingredientId)

	currentOwnerBytes, err = json.Marshal(currentOwner)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(currentOwnerId), currentOwnerBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 插入食材变更记录
	history := &IngredientHistory{
		IngredientId:   ingredientId,
		OriginOwnerId:  ownerId,
		CurrentOwnerId: currentOwnerId,
	}
	historyBytes, err := json.Marshal(history)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal ingredient history error: %s", err))
	}

	historyKey, err := stub.CreateCompositeKey("history", []string{
		ingredientId,
		ownerId,
		currentOwnerId,
	})
	if err != nil {
		return shim.Error(fmt.Sprintf("create key error: %s", err))
	}

	if err := stub.PutState(historyKey, historyBytes); err != nil {
		return shim.Error(fmt.Sprintf("save ingredient history error: %s", err))
	}

	return shim.Success(nil)
}

// 食材变更
func (c *IngredientsExchangeCC) ingredientExchangeFood(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//检查参数的个数
	if len(args) != 3 {
		return shim.Error("not enough args")
	}

	//验证参数的正确性
	ownerId := args[0]
	ingredientId := args[1]
	currentOwnerId := args[2]
	if ownerId == "" || ingredientId == "" || currentOwnerId == "" {
		return shim.Error("invalid args")
	}

	//验证数据是否存在
	originOwnerBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(originOwnerBytes) == 0 {
		return shim.Error("user not found")
	}

	currentOwnerBytes, err := stub.GetState(constructFOODKey(currentOwnerId))
	if err != nil || len(currentOwnerBytes) == 0 {
		return shim.Error("user not found")
	}

	assetBytes, err := stub.GetState(constructIngredientKey(ingredientId))
	if err != nil || len(assetBytes) == 0 {
		return shim.Error("ingredient not found")
	}

	// 校验原始拥有者确实拥有当前变更的食材
	originOwner := new(User)
	// 反序列化用户
	if err := json.Unmarshal(originOwnerBytes, originOwner); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	aidexist := false
	for _, aid := range originOwner.Ingredients {
		if aid == ingredientId {
			aidexist = true
			break
		}
	}
	if !aidexist {
		return shim.Error("ingredient owner not match")
	}

	//写入状态
	ingredientIds := make([]string, 0)
	for _, aid := range originOwner.Ingredients {
		if aid == ingredientId {
			continue
		}

		ingredientIds = append(ingredientIds, aid)
	}
	originOwner.Ingredients = ingredientIds

	originOwnerBytes, err = json.Marshal(originOwner)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(ownerId), originOwnerBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 当前拥有者插入食材id
	currentOwner := new(Food)
	// 反序列化用户
	if err := json.Unmarshal(currentOwnerBytes, currentOwner); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	currentOwner.Ingredients = append(currentOwner.Ingredients, ingredientId)

	currentOwnerBytes, err = json.Marshal(currentOwner)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructFOODKey(currentOwnerId), currentOwnerBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 插入食材变更记录
	history := &IngredientHistory{
		IngredientId:   ingredientId,
		OriginOwnerId:  ownerId,
		CurrentOwnerId: currentOwnerId,
	}
	historyBytes, err := json.Marshal(history)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal ingredient history error: %s", err))
	}

	historyKey, err := stub.CreateCompositeKey("history", []string{
		ingredientId,
		ownerId,
		currentOwnerId,
	})
	if err != nil {
		return shim.Error(fmt.Sprintf("create key error: %s", err))
	}

	if err := stub.PutState(historyKey, historyBytes); err != nil {
		return shim.Error(fmt.Sprintf("save ingredient history error: %s", err))
	}

	return shim.Success(nil)
}

// 外卖变更
func (c *IngredientsExchangeCC) foodExchange(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//检查参数的个数
	if len(args) != 3 {
		return shim.Error("not enough args")
	}

	//验证参数的正确性
	ownerId := args[0]
	foodId := args[1]
	currentOwnerId := args[2]
	if ownerId == "" || foodId == "" || currentOwnerId == "" {
		return shim.Error("invalid args")
	}

	//验证数据是否存在
	originOwnerBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(originOwnerBytes) == 0 {
		return shim.Error("user not found")
	}

	currentOwnerBytes, err := stub.GetState(constructUserKey(currentOwnerId))
	if err != nil || len(currentOwnerBytes) == 0 {
		return shim.Error("user not found")
	}

	foodBytes, err := stub.GetState(constructFoodKey(foodId))
	if err != nil || len(foodBytes) == 0 {
		return shim.Error("food not found")
	}

	// 校验原始拥有者确实拥有当前变更的外卖
	originOwner := new(User)
	// 反序列化用户
	if err := json.Unmarshal(originOwnerBytes, originOwner); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	aidexist := false
	for _, aid := range originOwner.Foods {
		if aid == foodId {
			aidexist = true
			break
		}
	}
	if !aidexist {
		return shim.Error("food owner not match")
	}

	//写入状态
	foodIds := make([]string, 0)
	for _, aid := range originOwner.Foods {
		if aid == foodId {
			continue
		}

		foodIds = append(foodIds, aid)
	}
	originOwner.Foods = foodIds

	originOwnerBytes, err = json.Marshal(originOwner)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(ownerId), originOwnerBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 当前拥有者插入外卖id
	currentOwner := new(User)
	// 反序列化用户
	if err := json.Unmarshal(currentOwnerBytes, currentOwner); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	currentOwner.Foods = append(currentOwner.Foods, foodId)

	currentOwnerBytes, err = json.Marshal(currentOwner)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(currentOwnerId), currentOwnerBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 插入外卖变更记录
	history := &FoodHistory{
		FoodId:         foodId,
		OriginOwnerId:  ownerId,
		CurrentOwnerId: currentOwnerId,
	}
	historyBytes, err := json.Marshal(history)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal food history error: %s", err))
	}

	historyKey, err := stub.CreateCompositeKey("history", []string{
		foodId,
		ownerId,
		currentOwnerId,
	})
	if err != nil {
		return shim.Error(fmt.Sprintf("create key error: %s", err))
	}

	if err := stub.PutState(historyKey, historyBytes); err != nil {
		return shim.Error(fmt.Sprintf("save food history error: %s", err))
	}

	return shim.Success(nil)
}

// 用户查询
func (c *IngredientsExchangeCC) queryUser(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//检查参数的个数
	if len(args) != 1 {
		return shim.Error("not enough args")
	}

	//验证参数的正确性
	ownerId := args[0]
	if ownerId == "" {
		return shim.Error("invalid args")
	}

	//验证数据是否存在
	userBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(userBytes) == 0 {
		return shim.Error("user not found")
	}

	return shim.Success(userBytes)
}

// 食材查询
func (c *IngredientsExchangeCC) queryIngredient(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//检查参数的个数
	if len(args) != 1 {
		return shim.Error("not enough args")
	}

	//验证参数的正确性
	ingredientId := args[0]
	if ingredientId == "" {
		return shim.Error("invalid args")
	}

	//验证数据是否存在
	ingredientBytes, err := stub.GetState(constructIngredientKey(ingredientId))
	if err != nil || len(ingredientBytes) == 0 {
		return shim.Error("ingredient not found")
	}

	return shim.Success(ingredientBytes)
}

//外卖查询
func (c *IngredientsExchangeCC) queryFood(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//检查参数的个数
	if len(args) != 1 {
		return shim.Error("not enough args")
	}

	//验证参数的正确性
	foodId := args[0]
	if foodId == "" {
		return shim.Error("invalid args")
	}

	//验证数据是否存在
	foodBytes, err := stub.GetState(constructFoodKey(foodId))
	if err != nil || len(foodBytes) == 0 {
		return shim.Error("food not found")
	}

	return shim.Success(foodBytes)
}

// 食材变更历史查询
func (c *IngredientsExchangeCC) queryIngredientHistory(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//检查参数的个数
	if len(args) != 2 && len(args) != 1 {
		return shim.Error("not enough args")
	}

	//验证参数的正确性
	ingredientId := args[0]
	if ingredientId == "" {
		return shim.Error("invalid args")
	}

	queryType := "all"
	if len(args) == 2 {
		queryType = args[1]
	}

	if queryType != "all" && queryType != "enroll" && queryType != "exchange" {
		return shim.Error(fmt.Sprintf("queryType unknown %s", queryType))
	}

	//验证数据是否存在
	ingredientBytes, err := stub.GetState(constructIngredientKey(ingredientId))
	if err != nil || len(ingredientBytes) == 0 {
		return shim.Error("ingredient not found")
	}

	// 查询相关数据
	keys := make([]string, 0)
	keys = append(keys, ingredientId)
	switch queryType {
	case "enroll":
		keys = append(keys, originOwner)
	case "exchange", "all":
	default:
		return shim.Error(fmt.Sprintf("unsupport queryType: %s", queryType))
	}
	result, err := stub.GetStateByPartialCompositeKey("history", keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("query history error: %s", err))
	}
	defer result.Close()

	histories := make([]*IngredientHistory, 0)
	for result.HasNext() {
		historyVal, err := result.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("query error: %s", err))
		}

		history := new(IngredientHistory)
		if err := json.Unmarshal(historyVal.GetValue(), history); err != nil {
			return shim.Error(fmt.Sprintf("unmarshal error: %s", err))
		}

		// 过滤掉不是食材转让的记录
		if queryType == "exchange" && history.OriginOwnerId == originOwner {
			continue
		}

		histories = append(histories, history)
	}

	historiesBytes, err := json.Marshal(histories)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal error: %s", err))
	}

	return shim.Success(historiesBytes)
}

// 食材变更历史查询
func (c *IngredientsExchangeCC) queryFoodHistory(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//检查参数的个数
	if len(args) != 2 && len(args) != 1 {
		return shim.Error("not enough args")
	}

	//验证参数的正确性
	foodId := args[0]
	if foodId == "" {
		return shim.Error("invalid args")
	}

	queryType := "all"
	if len(args) == 2 {
		queryType = args[1]
	}

	if queryType != "all" && queryType != "enroll" && queryType != "exchange" {
		return shim.Error(fmt.Sprintf("queryType unknown %s", queryType))
	}

	//验证数据是否存在
	foodBytes, err := stub.GetState(constructFoodKey(foodId))
	if err != nil || len(foodBytes) == 0 {
		return shim.Error("food not found")
	}

	// 查询相关数据
	keys := make([]string, 0)
	keys = append(keys, foodId)
	switch queryType {
	case "enroll":
		keys = append(keys, originOwner)
	case "exchange", "all":
	default:
		return shim.Error(fmt.Sprintf("unsupport queryType: %s", queryType))
	}
	result, err := stub.GetStateByPartialCompositeKey("history", keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("query history error: %s", err))
	}
	defer result.Close()

	histories := make([]*FoodHistory, 0)
	for result.HasNext() {
		historyVal, err := result.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("query error: %s", err))
		}

		history := new(FoodHistory)
		if err := json.Unmarshal(historyVal.GetValue(), history); err != nil {
			return shim.Error(fmt.Sprintf("unmarshal error: %s", err))
		}

		// 过滤掉不是食材转让的记录
		if queryType == "exchange" && history.OriginOwnerId == originOwner {
			continue
		}

		histories = append(histories, history)
	}

	historiesBytes, err := json.Marshal(histories)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal error: %s", err))
	}

	return shim.Success(historiesBytes)
}

func (c *IngredientsExchangeCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (c *IngredientsExchangeCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	funcName, args := stub.GetFunctionAndParameters()

	switch funcName {
	case "userRegister":
		return c.userRegister(stub, args)
	case "userDestroy":
		return c.userDestroy(stub, args)
	case "ingredientEnroll":
		return c.ingredientEnroll(stub, args)
	case "foodEnroll":
		return c.foodEnroll(stub, args)
	case "ingredientExchange":
		return c.ingredientExchange(stub, args)
	case "foodExchange":
		return c.foodExchange(stub, args)
	case "ingredientExchangeFood":
		return c.ingredientExchangeFood(stub, args)
	case "queryUser":
		return c.queryUser(stub, args)
	case "queryIngredient":
		return c.queryIngredient(stub, args)
	case "queryFood":
		return c.queryFood(stub, args)
	case "queryIngredientHistory":
		return c.queryIngredientHistory(stub, args)
	case "queryFoodHistory":
		return c.queryFoodHistory(stub, args)
	default:
		return shim.Error(fmt.Sprintf("unsupported function: %s", funcName))
	}

}
func main() {
	err := shim.Start(new(IngredientsExchangeCC))
	if err != nil {
		fmt.Printf("Error starting AssertsExchange chaincode: %s", err)
	}
}
