package main

import (
  "net/http"
  "io/ioutil"
  "encoding/json"
  "encoding/csv"
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

func WriteJSON(writer http.ResponseWriter, result interface{}){
  jsonResult, err := json.MarshalIndent(result, "", "  ")
  if err != nil {
    writer.Write([]byte(err.Error()))
  }
  writer.Write(jsonResult) 
}

func WriteCSV(writer http.ResponseWriter, result []bson.M) {
  data := make([][]string, 0)
  headers := make([]string, 0)
  for key, _ := range result[0] {
    headers = append(headers, key)
  }
  data = append(data, headers)
  // each map[string]interface{} in result
  for _, item := range result {
    row := make([]string, 0)
    // each key/value pair in item
    data = append(data, row)
    for _, value := range item {
      if point, ok := value.([]interface{}); ok {
        row = append(row, point[0].(string))
      } else if point, ok := value.(string); ok {
        row = append(row, point)
      } else if point, ok := value.(bson.ObjectId); ok {
        row = append(row, point.String())
      }
    }
    data = append(data, row)
  }
  csvWriter := csv.NewWriter(writer)
  err := csvWriter.WriteAll(data)
  if err != nil {
    writer.Write([]byte(err.Error()))
  }
}

type QuestionIndexHandler struct{}
func (q QuestionIndexHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  vars := mux.Vars(request)
  extension := vars["extension"]
  c := session.DB("oqsurvey").C("questions")
  var resultArray []bson.M
  err := c.Find(bson.M{}).All(&resultArray)
  if err != nil {
    writer.Write([]byte(err.Error()))
  }
  if extension == ".csv" {
    WriteCSV(writer, resultArray)
  } else {
    WriteJSON(writer, resultArray)
  }
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
  WriteJSON(writer, form)
}

type AnswerIndexHandler struct{}
func (a AnswerIndexHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  vars := mux.Vars(request)
  id := vars["id"];
  extension := vars["extension"]
  questionId := make([]string, 1)
  questionId[0] = id;
  c := session.DB("oqsurvey").C("answers")
  var resultArray []bson.M
  err := c.Find(bson.M{"question_id": questionId}).All(&resultArray);
  if err != nil {
    writer.Write([]byte(err.Error()))
  }
  if extension == ".csv" {
    WriteCSV(writer, resultArray)
  } else {
    // json
    WriteJSON(writer, resultArray)
  }
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
  router.Handle("/questions{extension}", QuestionsHandler)
  router.Handle("/questions/{id}/answers", AnswersHandler)
  router.Handle("/questions/{id}/answers{extension}", AnswersHandler) 
  http.Handle("/", router)
  http.ListenAndServe("localhost:4000", nil)
}