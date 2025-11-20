package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"

	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ahmedjebari022/Chripy/internal/auth"
	"github.com/ahmedjebari022/Chripy/internal/database"
	"github.com/google/uuid"
)

const PORT = "8080"

func Start(){
	mutex := http.NewServeMux()
	dbUrl := os.Getenv("DB_URL")
	secretKey := os.Getenv("SECRET_KEY")
	polkaApiKey := os.Getenv("POLKA_KEY")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)

	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db: dbQueries,
		sk: secretKey,
		pk: polkaApiKey,
	}
	server := http.Server{
		Addr : ":" + PORT,
		Handler: mutex,
	}


	mutex.Handle("/app/",cfg.middlewareMetricsInc(http.StripPrefix("/app/",http.FileServer(http.Dir(".")))))
	mutex.HandleFunc("GET /api/healthz",handlerHealthz)
	mutex.HandleFunc("GET /admin/metrics",cfg.handlerMetrics)
	mutex.HandleFunc("POST /admin/reset",cfg.handlerReset)
	mutex.HandleFunc("POST /api/validate_chirp",handlerValidateChirp)
	mutex.HandleFunc("POST /api/users",cfg.handlerCreateUser)
	mutex.HandleFunc("POST /api/chirps",cfg.HandlerCreateChirps)
	mutex.HandleFunc("GET /api/chirps",cfg.HandlerGetChirps)
	mutex.HandleFunc("GET /api/chirps/{chirpId}",cfg.HandlerGetChirp)
	mutex.HandleFunc("POST /api/login",cfg.HandlerLogin)
	mutex.HandleFunc("POST /api/refresh",cfg.HandlerRefresh)
	mutex.HandleFunc("POST /api/revoke",cfg.HandlerRevoke)
	mutex.HandleFunc("PUT /api/users",cfg.HandlerUpdateCred)
	mutex.HandleFunc("DELETE /api/chirps/{chirpId}",cfg.handlerDeleteChirp)
	mutex.HandleFunc("POST /api/polka/webhooks",cfg.HandlerPolkaWebhook)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Server listening on port: %s",PORT)
}

func handlerHealthz(w http.ResponseWriter,req *http.Request){
	
	w.Header().Set("Content-Type","text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
	
}

type apiConfig struct{
	fileserverHits atomic.Int32
	db *database.Queries
	sk string
	pk string
}


func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler{
	return http.HandlerFunc(func(w http.ResponseWriter,req *http.Request){
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	})
	
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter,req *http.Request){
	w.Header().Set("Content-Type","text/html; charset=utf-8")
	body := fmt.Sprintf(`<html>
							<body>
								<h1>Welcome, Chirpy Admin</h1>
								<p>Chirpy has been visited %d times!</p>
							</body>
						</html>`,cfg.fileserverHits.Load())

	w.WriteHeader(200)
	w.Write([]byte(body))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, req *http.Request){
	
	w.Header().Set("Content-Type","text/plain: charset=utf-8")
	cfg.fileserverHits.Store(0)
	w.WriteHeader(200)
}


func handlerValidateChirp(w http.ResponseWriter, req *http.Request){
	
	type requestBody struct{
		Body string `json:"body"`
		
	}
	
	decoder := json.NewDecoder(req.Body)
	var reqBody requestBody
	err := decoder.Decode(&reqBody)
	if err != nil{
		respondeWithError(w,500,"Something went wrong")
		return
	}
	if len(reqBody.Body) > 140{
		respondeWithError(w,400,"Chirp is too long")
		return
	}
	if reqBody.Body == ""{
		respondeWithError(w,400,"body field is required")
		return
	}
	profaneWords := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}
	splitedBody := strings.Split(reqBody.Body," ")
	for i, sb := range splitedBody{
		for _, pw := range profaneWords{
			if strings.ToLower(sb) == pw{
				splitedBody[i] = "****"
			}
		}

	}
	cleanedBody := strings.Join(splitedBody," ")
	fmt.Printf("%s\n",cleanedBody)
	response := struct {
			CleanedBody string `json:"cleaned_body"`
		}{
			CleanedBody: cleanedBody,
		}
	respondeWithJson(w,200,response)
}

type User struct{
	Id  uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"` 
	UpdateAt time.Time `json:"updated_at"`
	Email string `json:"email"`
	IsChirpyRed bool `json:"is_chirpy_red"`
}

type Chirp struct{
	Id uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"` 
	UpdateAt time.Time `json:"updated_at"`
	Body string `json:"body"`
	UserId uuid.UUID `json:"user_id"`
	
}

type Chirps struct{
	Items []Chirp `json:"chirps"`
}


func (cfg *apiConfig)handlerCreateUser(w http.ResponseWriter, req *http.Request){
	
	type requestBody  struct{
		Email string `json:"email"`
		Password string `json:"password"`
	}
	var reqBody  requestBody
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&reqBody)
	if err != nil {
		respondeWithError(w,400,fmt.Sprintf("error: %s",err.Error()))
		return
	}
	if reqBody.Email == ""{
		respondeWithError(w,400,"missing email in body")
	}
	hashed, err := auth.HashPassword(reqBody.Password)
	if err != nil{
		respondeWithError(w,500,err.Error())
	}
	user, err := cfg.db.CreateUser(req.Context(),database.CreateUserParams{
		Email: reqBody.Email,
		HashedPassword: hashed,
	})
	if err != nil {
		respondeWithError(w,400,fmt.Sprintf("error: %s",err.Error()))
		return
	}
	res := User{
		Id: user.ID,
		CreatedAt: user.CreatedAt,
		UpdateAt: user.UpdatedAt,
		Email: user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}
	err = respondeWithJson(w,201,res)
	if err != nil {
		respondeWithError(w,400,fmt.Sprintf("error: %s",err.Error()))
	}
	
}
//helper functions 

func (cfg *apiConfig)HandlerCreateChirps(w http.ResponseWriter, r *http.Request){
		type reqBody struct{
			Body string `json:"body"`
			UserId uuid.UUID `json:"user_id"`
		}
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondeWithError(w,401,err.Error())
			return 
		}
		id, err := auth.ValidateJWT(token, cfg.sk)
		if err != nil {
			respondeWithError(w, 401, err.Error())
			return 
		}
		decoder := json.NewDecoder(r.Body)
		var req reqBody
		err = decoder.Decode(&req)
		if err != nil {
			w.Header().Set("Content-Type","application/json")
			w.WriteHeader(400)
			errBody := struct{
				Error string `json:"error"`
			}{
				Error: fmt.Sprintf("error: %s\n",err.Error()),
			}
			decodedErr, err:= json.Marshal(errBody)
			if err != nil {
				w.Write([]byte(`{"error":"internal server error"}`))
				return
			}
			w.Write(decodedErr)
			return
		}
		if len(req.Body) > 140{
			err = respondeWithError(w,400,"maximaume length exceeded")
			if err != nil {
				w.Header().Set("Content-Type","application/json")
				w.WriteHeader(500)
       			w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
				return
			}
			return
		}
		

		chirp, err := cfg.db.CreateChirp(r.Context(),database.CreateChirpParams{
			Body: req.Body,
			UserID: id,
		})
		if err != nil {
			respondeWithError(w,500,err.Error())
			return
		}
		chirpResponse := Chirp{
			Id: chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdateAt: chirp.UpdatedAt,
			Body: chirp.Body,
			UserId: id,
		}
		err = respondeWithJson(w,201,chirpResponse)
		if err != nil {
			w.Header().Set("Content-Type","application/json")
				w.WriteHeader(500)
       			w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
				return	
		}
}
func parseQuery(sp string) (uuid.NullUUID,error){
	if sp == ""{
		return uuid.NullUUID{Valid: false},nil
	}
	
	uid, err := uuid.Parse(sp)
	if err != nil {
		return uuid.NullUUID{},err
	}
	return uuid.NullUUID{Valid: true,UUID: uid},nil

}
func (cfg *apiConfig) HandlerGetChirps(w http.ResponseWriter,r *http.Request){
	qpId := r.URL.Query().Get("author_id")
	qpSort := r.URL.Query().Get("sort")

	authorUuid, err := parseQuery(qpId)
	if err != nil {
		respondeWithError(w,500,err.Error())
		return
	}
	fmt.Printf("%v",authorUuid)
	chirps, err := cfg.db.GetChirps(r.Context(),authorUuid)
	if err != nil {
		respondeWithError(w,500,err.Error())
		return
	}
	resChirps := Chirps{
		Items: make([]Chirp,len(chirps)),
		}
	for i, c := range chirps {
		chirp := Chirp{
			Id: c.ID,
			CreatedAt: c.CreatedAt,
			UpdateAt: c.UpdatedAt,
			Body: c.Body,
			UserId: c.UserID,
		}
		resChirps.Items[i] = chirp
	}
	if qpSort == "desc"{
		slices.Reverse(resChirps.Items)
	}
	respondeWithJson(w,200,resChirps.Items)	
}


func (cfg *apiConfig) HandlerGetChirp(w http.ResponseWriter,req *http.Request){
	
	id := req.PathValue("chirpId")
	chirpId, err := uuid.Parse(id)
	if err != nil {
		respondeWithError(w,500,err.Error())
		return
	} 
	chirp, err := cfg.db.GetChirp(req.Context(),chirpId)
	if err != nil {
		respondeWithError(w,404,err.Error())
	}
	chirpRes := Chirp{
		Id: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdateAt: chirp.UpdatedAt,
		Body: chirp.Body,
		UserId: chirp.UserID,
	}
	respondeWithJson(w,200,chirpRes)
}

func (cfg *apiConfig) HandlerRefresh(w http.ResponseWriter, req *http.Request){
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondeWithError(w,401,err.Error())
		return
	}
	refToken, err := cfg.db.GetRefreshTokenByToken(req.Context(),token)
	if err != nil {
		respondeWithError(w,401,err.Error())
		return
	}
	user, err := cfg.db.GetUserFromRefreshToken(req.Context(),refToken.Token)
	if err != nil {
		respondeWithError(w,401,err.Error())
		return
	}
	accToken, err := auth.MakeJWT(user.ID,cfg.sk,time.Hour)
	if err != nil{
		respondeWithError(w,500,err.Error())
		return
	}
	var resToken = struct{
		Token string `json:"token"`
	}{
		Token: accToken,
	}
	respondeWithJson(w,200,resToken)
}




func (cfg *apiConfig) HandlerLogin(w http.ResponseWriter,req *http.Request){
	type loginRequest struct{
		Email string `json:"email"`
		Password string `json:"password"`
		ExpiresIn time.Duration `json:"expires_in_seconds"`
	}
	decoder := json.NewDecoder(req.Body)
	var loginReq loginRequest
	err := decoder.Decode(&loginReq)
	if err != nil {
		respondeWithError(w,400,err.Error())
		return 
	}
	user, err := cfg.db.GetUserByEmail(req.Context(),loginReq.Email)
	if err != nil {
		respondeWithError(w,401,"Incorrect email or password")
		return
	}
	match, err := auth.CheckPasswordHash(loginReq.Password,user.HashedPassword)
	if err != nil {
		respondeWithError(w,500,err.Error())
		return
	}
	if !match {
		respondeWithError(w,401,"Incoorect email or password")
		return 
	}
	var duration time.Duration = time.Hour
	token, _ := auth.MakeRefreshToken()
	exp, _ := cfg.db.CreateToken(req.Context(),database.CreateTokenParams{
		Token: token,
		UserID: user.ID,
		ExpiresAt: time.Now().Add(60*24*time.Hour),
	})
	jwt, err := auth.MakeJWT(user.ID,cfg.sk,duration)
	if err != nil {
		respondeWithError(w,500,err.Error())
		return
	}
	userRes := struct{
		Id  uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"` 
		UpdateAt time.Time `json:"updated_at"`
		Email string `json:"email"`	
		Token string `json:"token"`
		RefreshToken string `json:"refresh_token"`
		IsChirpyRed bool `json:"is_chirpy_red"`
	}{
		Id: user.ID,
		CreatedAt: user.CreatedAt,
		UpdateAt: user.UpdatedAt,
		Email: user.Email,
		Token: jwt,
		RefreshToken: exp.Token,
		IsChirpyRed: user.IsChirpyRed,

	}
	respondeWithJson(w,200,userRes)
}


func (cfg *apiConfig)HandlerRevoke(w http.ResponseWriter,req *http.Request){
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondeWithError(w,401,err.Error())
		return
	}
	_, err = cfg.db.GetRefreshTokenByToken(req.Context(),token)
	if err != nil {
		respondeWithError(w,401,err.Error())
		return
	}
	err = cfg.db.RevokeToken(req.Context(),token)
	if err != nil {
		respondeWithError(w,500,err.Error())
		return
	}
	respondeWithJson(w,204,struct{}{})
}



func (cfg *apiConfig)HandlerUpdateCred(w http.ResponseWriter,req *http.Request){
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondeWithError(w,401,err.Error())
		return
	}
	userId, err := auth.ValidateJWT(token,cfg.sk)
	if err != nil {
		respondeWithError(w,401,err.Error())
		return
	}
	decoder := json.NewDecoder(req.Body)
	
	type reqStruct struct{
		Email string `json:"email"`
		Password string `json:"password"`
	}
	var reqBody reqStruct 
	err = decoder.Decode(&reqBody)
	if err != nil {
		respondeWithError(w,400,err.Error())
		return
	}
	hashed, err := auth.HashPassword(reqBody.Password)
	if err != nil {
		respondeWithError(w,500,err.Error())
		return
	}
	user ,err := cfg.db.UpdateUserById(req.Context(),database.UpdateUserByIdParams{
		Email: reqBody.Email,
		HashedPassword: hashed,
		ID: userId,
	})
	if err != nil {
		respondeWithError(w,500,err.Error())
		return 
	}
	resUser := User{
		Id: user.ID,
		Email: user.Email,
		CreatedAt: user.CreatedAt,
		UpdateAt: user.UpdatedAt,
		IsChirpyRed: user.IsChirpyRed,
	}
	respondeWithJson(w,200,resUser)
}

func (cfg *apiConfig)handlerDeleteChirp(w http.ResponseWriter,req *http.Request){
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondeWithError(w,401,err.Error())
		return
	}
	userId, err := auth.ValidateJWT(token,cfg.sk)
	if err != nil {
		respondeWithError(w,401,err.Error())
		return
	}
	pId := req.PathValue("chirpId")
	chirpId, _ := uuid.Parse(pId)
	chirp, err := cfg.db.GetChirp(req.Context(),chirpId)
	if err != nil {
		respondeWithError(w,404,err.Error())
		return
	}
	if chirp.UserID != userId{
		respondeWithError(w,403,"unauthorized user")
		return
	}
	err = cfg.db.DeleteChirpById(req.Context(),chirp.ID)
	if err != nil {
		respondeWithError(w,500,err.Error())
		return
	}
	respondeWithJson(w,204,struct{}{})
}

func (cfg *apiConfig)HandlerPolkaWebhook(w http.ResponseWriter,req * http.Request){
	apiKey, err := auth.GetAPIKey(req.Header)
	if err != nil {
		respondeWithError(w,401,err.Error())
		return 
	}
	fmt.Printf("api_key%s\n",apiKey)
	if apiKey != cfg.pk {
		respondeWithError(w,401,"api keys don't match")
		return
	}
	


	type reqBody struct{
		Event string `json:"event"`
		Data struct{
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}	
	decoder := json.NewDecoder(req.Body)
	var r reqBody
	err = decoder.Decode(&r)
	if err != nil {
		respondeWithError(w,400,err.Error())
		return
	}
	if r.Event != "user.upgraded"{
		respondeWithJson(w,204,struct{}{})
		return
	}
	err = cfg.db.UpgradeUserToChirpy(req.Context(),r.Data.UserID)
	if err != nil {
		respondeWithError(w,404,err.Error())		
		return
	}
	respondeWithJson(w,204,struct{}{})
}






func respondeWithJson(w http.ResponseWriter, code int, payload any)error{
	response, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(code)
	if code == 204{
		return nil
	}
	w.Write(response)	
	return nil
}

func respondeWithError(w http.ResponseWriter, code int , msg string)error{
	return respondeWithJson(w,code,map[string]string{"error":msg})
}