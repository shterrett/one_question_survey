package main

import (
  "net/http"
  "io/ioutil"
  "encoding/json"
  "strings"
  "github.com/gorilla/mux"
  "github.com/gorilla/handlers"
  "labix.org/v2/mgo"
  "labix.org/v2/mgo/bson"
)

var session *mgo.Session
var sessionError error

func NewQuestionHandler(writer http.ResponseWriter, request *http.Request) {
  file, err := ioutil.ReadFile("./questions/new.html")
  if err != nil {
    writer.Write([]byte(err.Error()))
  } else {
    writer.Write(file)
  }
}

type QuestionIndexHandler struct{}
func (q QuestionIndexHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  c := session.DB("oqsurvey").C("questions")
  var resultArray []bson.M
  err := c.Find(bson.M{}).All(&resultArray)
  if err != nil {
    writer.Write([]byte(err.Error()))
  }
  jsonResult, err := json.MarshalIndent(resultArray, "", "  ")
  if err != nil {
    writer.Write([]byte(err.Error()))
  }
  writer.Write(jsonResult)  
  // Query all questions from database; present in JSON and CSV
}

type QuestionCreateHandler struct{}
func (q QuestionCreateHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  request.ParseForm()
  form := request.Form
  c := session.DB("oqsurvey").C("questions")
  err := c.Insert(form)
  if err != nil {
    writer.Write([]byte(err.Error()))
  } else {
    for key, value := range form {
      writer.Write([]byte("Question Successfully Created\n"))
      writer.Write([]byte(key))
      writer.Write([]byte(" | "))
      writer.Write([]byte(value[0]))
    }
  }
}

type AnswerCreateHandler struct{}
func (a AnswerCreateHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  writer.Write([]byte("New Answer"))
  url := request.URL.Path
  urlParts := strings.Split(url, "/")
  request.ParseForm()
  form := request.Form
  form.Add("question_id", urlParts[2])
  c := session.DB("oqsurvey").C("answers")
  err := c.Insert(form)
  if err != nil {
    writer.Write([]byte(err.Error()))
  }
  jsonAnswer, err := json.MarshalIndent(form, "", "  ")
  writer.Write(jsonAnswer)
  // Write Answer to database. Status code only; no return
}

type AnswerIndexHandler struct{}
func (a AnswerIndexHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  url := request.URL.Path
  urlParts := strings.Split(url, "/")
  questionId := make([]string, 1)
  questionId[0] = urlParts[2];
  c := session.DB("oqsurvey").C("answers")
  var resultArray []bson.M
  err := c.Find(bson.M{"question_id": questionId}).All(&resultArray);
  if err != nil {
    writer.Write([]byte(err.Error()))
  }
  jsonAnswers, err := json.MarshalIndent(resultArray, "", "  ")
  if err != nil {
    writer.Write([]byte(err.Error()))
  }
  writer.Write(jsonAnswers)
  // Read all answers from database (for question). Return as csv or JSON
}

func main() {
  session, sessionError = mgo.Dial("localhost")
  if sessionError != nil {
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
  router.HandleFunc("/questions/new", NewQuestionHandler) // consider Filesystem handler so that css etc is also served
  router.Handle("/questions", QuestionsHandler)
  router.Handle("/questions/{id}/answers", AnswersHandler) 
  http.Handle("/", router)
  http.ListenAndServe("localhost:4000", nil)
}