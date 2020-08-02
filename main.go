package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/twinj/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

//Person ...
type Person struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	UUID      uuid.UUID          `json:"uuid,omitempty" bson:"uuid,omitempty"`
	Firstname string             `json:"firstname,omitempty" bson:"firstname,omitempty"`
}

type TokenDetails struct {
	AccessToken  string
	RefreshToken string
	AccessUUID   string
	RefreshUUID  string
	AtExpires    int64
	RtExpires    int64
}

type tokensfromdb struct {
	ID           primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	UserID       string             `json:"user_id,omitempty" bson:"user_id,omitempty"`
	AccessUUID   string             `json:"access_uuid,omitempty" bson:"access_uuid,omitempty"`
	RefreshUUID  string             `json:"refresh_uuid,omitempty" bson:"refresh_uuid,omitempty"`
	AccessToken  string             `json:"access_token,omitempty" bson:"access_token,omitempty"`
	RefreshToken string             `json:"refresh_token,omitempty" bson:"refresh_token,omitempty"`
}

func PostUserEndpoint(response http.ResponseWriter, request *http.Request) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb+srv://dbAdmin:yzEUyKDAKPHg8HNE@cluster0.fen7o.mongodb.net/MEDODS?retryWrites=true&w=majority"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	usersCollection := client.Database("MEDODS").Collection("users")

	response.Header().Set("content-type", "application/json")
	var person Person
	_ = json.NewDecoder(request.Body).Decode(&person)
	person.UUID = uuid.NewV4()
	result, _ := usersCollection.InsertOne(ctx, person)
	json.NewEncoder(response).Encode(result)
}

func GetUsersEndpoint(response http.ResponseWriter, request *http.Request) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb+srv://dbAdmin:yzEUyKDAKPHg8HNE@cluster0.fen7o.mongodb.net/MEDODS?retryWrites=true&w=majority"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	usersCollection := client.Database("MEDODS").Collection("users")

	cursor, _ := usersCollection.Find(ctx, bson.M{})
	var people []Person
	for cursor.Next(ctx) {
		var person Person
		cursor.Decode(&person)
		people = append(people, person)
	}
	json.NewEncoder(response).Encode(people)
}

func CreateToken(userid string) (*TokenDetails, error) {
	td := &TokenDetails{}
	td.AtExpires = time.Now().Add(time.Minute * 15).Unix()
	td.AccessUUID = uuid.NewV4().String()

	td.RtExpires = time.Now().Add(time.Hour * 24 * 7).Unix()
	td.RefreshUUID = uuid.NewV4().String()

	var err error
	//Creating Access Token
	os.Setenv("ACCESS_SECRET", "jdnfksdmfksd") //this should be in an env file
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["access_uuid"] = td.AccessUUID
	atClaims["user_id"] = userid
	atClaims["exp"] = td.AtExpires
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	td.AccessToken, err = at.SignedString([]byte(os.Getenv("ACCESS_SECRET")))
	if err != nil {
		return nil, err
	}
	//Creating Refresh Token
	os.Setenv("REFRESH_SECRET", "mcmvmkmsdnfsdmfdsjf") //this should be in an env file
	rtClaims := jwt.MapClaims{}
	rtClaims["refresh_uuid"] = td.RefreshUUID
	rtClaims["user_id"] = userid
	rtClaims["exp"] = td.RtExpires
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = rt.SignedString([]byte(os.Getenv("REFRESH_SECRET")))
	if err != nil {
		return nil, err
	}

	//Связываем acess и refresh токены
	td.RefreshToken = td.RefreshToken + td.AccessToken[len(td.AccessToken)-6:]

	return td, nil
}

func savetokens(td *TokenDetails, userid string) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb+srv://dbAdmin:yzEUyKDAKPHg8HNE@cluster0.fen7o.mongodb.net/MEDODS?retryWrites=true&w=majority"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	usersCollection := client.Database("MEDODS").Collection("tokens")

	//Кодируем в bcrypt
	bytes, _ := bcrypt.GenerateFromPassword([]byte(td.RefreshToken), 10)

	tokens := map[string]interface{}{
		"user_id":       userid,
		"access_uuid":   td.AccessUUID,
		"refresh_uuid":  td.RefreshUUID,
		"access_token":  td.AccessToken,
		"refresh_token": bytes,
	}
	json.Marshal(&tokens)

	usersCollection.InsertOne(ctx, tokens)
}

func GetTokensEndpoint(response http.ResponseWriter, request *http.Request) {
	params := mux.Vars(request)
	id := params["id"]

	token, err := CreateToken(id)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	savetokens(token, id)
	tokens := map[string]string{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
	}
	json.NewEncoder(response).Encode(tokens)
}

func RefreshTokensEndpoint(response http.ResponseWriter, request *http.Request) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb+srv://dbAdmin:yzEUyKDAKPHg8HNE@cluster0.fen7o.mongodb.net/MEDODS?retryWrites=true&w=majority"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	usersCollection := client.Database("MEDODS").Collection("tokens")

	params := mux.Vars(request)

	idd, _ := primitive.ObjectIDFromHex(params["id"])
	var tfdb tokensfromdb
	err = usersCollection.FindOne(ctx, tokensfromdb{ID: idd}).Decode(&tfdb)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}

	//Проверяем связаны ли токены
	//Так как мы храним refresh токен в хэшированном виде в базе, и обратной функции хэшированию нет,
	//то мы не можем получить исходный refresh токен. Так как мы его храним только в БД и больше нигде,
	//сравнить на связанность access и refresh токен не предоставляется возможным

	// if tfdb.AccessToken[len(tfdb.AccessToken)-6:] != tfdb.RefreshToken[len(tfdb.RefreshToken)-6:] {
	// 	response.Write([]byte(tfdb.AccessToken[len(tfdb.AccessToken)-6:]))
	// 	response.Write([]byte(tfdb.RefreshToken[len(tfdb.RefreshToken)-6:]))
	// 	response.Write([]byte(`Токены не совпадают`))
	// 	return
	// }

	//Генерируем новую пару
	newTokens, _ := CreateToken(tfdb.UserID)
	savetokens(newTokens, tfdb.UserID)

	bytes, _ := bcrypt.GenerateFromPassword([]byte(newTokens.RefreshToken), 10)

	update := bson.D{{"$set", bson.D{{"refresh_uuid", newTokens.RefreshUUID},
		{"access_uuid", newTokens.AccessUUID}, {"refresh_token", bytes},
		{"access_token", newTokens.AccessToken}}}}

	//Обновляем конкретную пару токенов
	err = usersCollection.FindOneAndUpdate(ctx, tokensfromdb{ID: idd}, update).Decode(&newTokens)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(response).Encode(newTokens)

}

func GetAllTokensEndpoint(response http.ResponseWriter, request *http.Request) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb+srv://dbAdmin:yzEUyKDAKPHg8HNE@cluster0.fen7o.mongodb.net/MEDODS?retryWrites=true&w=majority"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	usersCollection := client.Database("MEDODS").Collection("tokens")

	cursor, _ := usersCollection.Find(ctx, bson.M{})
	var tfbd []tokensfromdb
	for cursor.Next(ctx) {
		var tokenfromdb tokensfromdb
		cursor.Decode(&tokenfromdb)
		tfbd = append(tfbd, tokenfromdb)
	}
	json.NewEncoder(response).Encode(tfbd)
}

func DeleteTokenEndpoint(response http.ResponseWriter, request *http.Request) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb+srv://dbAdmin:yzEUyKDAKPHg8HNE@cluster0.fen7o.mongodb.net/MEDODS?retryWrites=true&w=majority"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	usersCollection := client.Database("MEDODS").Collection("tokens")

	params := mux.Vars(request)
	idd, _ := primitive.ObjectIDFromHex(params["id"])

	_, err = usersCollection.DeleteOne(ctx, tokensfromdb{ID: idd})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	response.Write([]byte(`Токен успешно удален`))
}

func DeleteAllTokenEndpoint(response http.ResponseWriter, request *http.Request) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb+srv://dbAdmin:yzEUyKDAKPHg8HNE@cluster0.fen7o.mongodb.net/MEDODS?retryWrites=true&w=majority"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	usersCollection := client.Database("MEDODS").Collection("tokens")

	params := mux.Vars(request)
	idd := params["id"]

	res, err := usersCollection.DeleteMany(ctx, tokensfromdb{UserID: idd})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(response).Encode(res)
}

func main() {
	fmt.Println("Start ...")

	router := mux.NewRouter()
	router.HandleFunc("/postuser", PostUserEndpoint).Methods("POST")
	router.HandleFunc("/users", GetUsersEndpoint).Methods("GET")
	router.HandleFunc("/tokens", GetAllTokensEndpoint).Methods("GET")
	router.HandleFunc("/gettokens/{id}", GetTokensEndpoint).Methods("GET")
	router.HandleFunc("/refreshtokens/{id}", RefreshTokensEndpoint).Methods("GET")
	router.HandleFunc("/deletetoken/{id}", DeleteTokenEndpoint).Methods("GET")
	router.HandleFunc("/deletealltoken/{id}", DeleteAllTokenEndpoint).Methods("GET")
	http.ListenAndServe(":9191", router)
}
