package main

import (
  "net/http"
  "io/ioutil"
  "github.com/gorilla/mux"
  "github.com/gorilla/handlers"
  "labix.org/v2/mgo"
)

var session *mgo.Session
var session_error error

func NewQuestionHandler(writer http.ResponseWriter, request *http.Request) {
  file, _ := ioutil.ReadFile("./questions/new.html")
  writer.Write(file)
}

type QuestionIndexHandler struct{}
func (q QuestionIndexHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  writer.Write([]byte("Index"))
  // Query all questions from database; present in JSON and CSV
}

type QuestionCreateHandler struct{}
func (q QuestionCreateHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  request.ParseForm()
  form := request.Form
  c := session.DB("oqsurvey").C("questions")
  c.Insert(form)
  for key, value := range form {
    writer.Write([]byte(key))
    writer.Write([]byte(" | "))
    writer.Write([]byte(value[0]))
  }
  // Write Question to database. Return success or failure
}

type AnswerCreateHandler struct{}
func (a AnswerCreateHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  writer.Write([]byte("New Answer"))
  // Write Answer to database. Status code only; no return
}

type AnswerIndexHandler struct{}
func (a AnswerIndexHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  writer.Write([]byte("Answers"))
  // Read all answers from database (for question). Return as csv or JSON
}

func main() {
  session, session_error = mgo.Dial("localhost")
  if session_error != nil {
    panic("Unable to connect to Database")
  }
  defer session.Close()
  QuestionsHandler := make(handlers.MethodHandler)
  QuestionsHandler["GET"] = QuestionIndexHandler{}
  QuestionsHandler["POST"] = QuestionCreateHandler{}
  AnswersHandler := make(handlers.MethodHandler)
  AnswersHandler["GET"] = AnswerIndexHandler{}
  AnswersHandler["POST"] = AnswerCreateHandler{}
  router := mux.NewRouter()
  router.HandleFunc("/questions/new", NewQuestionHandler)
  router.Handle("/questions", QuestionsHandler)
  router.Handle("/questions/{id}/answers", AnswersHandler)
  http.Handle("/", router)
  http.ListenAndServe("localhost:4000", nil)
}