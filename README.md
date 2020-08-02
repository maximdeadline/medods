# medods

В данном проекте реализовано 7 API маршрутов
1) router.HandleFunc("/postuser", PostUserEndpoint).Methods("POST") - добавление нового пользователя методом отправки JSON (только имя пользователя, _id и uuid генерируются автоматически)
2) router.HandleFunc("/users", GetUsersEndpoint).Methods("GET") - просмотр всех пользователей системы 
3) router.HandleFunc("/gettokens/{id}", GetTokensEndpoint).Methods("GET") - в параметре id передается uuid пользователя. В ответ к нему приходят два токена, и это все автоматически добавляется в БД
4) router.HandleFunc("/tokens", GetAllTokensEndpoint).Methods("GET") - просмотр всех токенов, которые есть в базе 
5) router.HandleFunc("/refreshtokens/{id}", RefreshTokensEndpoint).Methods("GET") - в параметре id передается _id пары токенов из бд, которые нужно обновить  
6) router.HandleFunc("/deletetoken/{id}", DeleteTokenEndpoint).Methods("GET") - в параметре id передается _id пары токенов из бд, которые нужно удалить
router.HandleFunc("/deletealltoken/{id}", DeleteAllTokenEndpoint).Methods("GET") - в параметре id передается UUID пользователя, для которого нужно удалить все токены из базы
