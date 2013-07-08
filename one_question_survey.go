package main

import (
  "net/http"
  "fmt"
  "io/ioutil"
  "encoding/json"
  "encoding/csv"
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
    fmt.Fprintf(writer, err.Error())
  } else {
    writer.Write(file)
  }
}

func WriteJSON(writer http.ResponseWriter, result interface{}){
  jsonResult, err := json.MarshalIndent(result, "", "  ")
  if err != nil {
    fmt.Fprintf(writer, err.Error())
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
    fmt.Fprintf(writer, err.Error())
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
    fmt.Fprintf(writer, err.Error())
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
    fmt.Fprintf(writer, err.Error())
  } else {
    for key, value := range form {
      fmt.Fprintf(writer, "Question Successfully Created\n")
      fmt.Fprintf(writer, key)
      fmt.Fprintf(writer, " | ")
      fmt.Fprintf(writer, value[0])
    }
  }
}

type AnswerCreateHandler struct{}
func (a AnswerCreateHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  fmt.Fprintf(writer, "New Answer")
  vars := mux.Vars(request)
  request.ParseForm()
  form := request.Form
  form.Add("question_id", vars["id"])
  c := session.DB("oqsurvey").C("answers")
  err := c.Insert(form)
  if err != nil {
    fmt.Fprintf(writer, err.Error())
  }
  WriteJSON(writer, form)
}

func GetAnswers(id string) ([]bson.M, error) {
  questionId := make([]string, 1)
  questionId[0] = id;
  c := session.DB("oqsurvey").C("answers")
  var resultArray []bson.M
  err := c.Find(bson.M{"question_id": questionId}).All(&resultArray);
  return resultArray, err
}

type AnswerIndexHandler struct{}
func (a AnswerIndexHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  vars := mux.Vars(request)
  extension := vars["extension"]
  resultArray, err := GetAnswers(vars["id"])
  if err != nil {
    fmt.Fprintf(writer, err.Error())
  }
  if extension == ".csv" {
    WriteCSV(writer, resultArray)
  } else {
    WriteJSON(writer, resultArray)
  }
}

func GetAnswerTotals(answers []bson.M) map[string]int {
  answerTotals := make(map[string]int)
  totalAnswers := 0
  for _, ans := range answers {
    answerAry := ans["answer"]
    if answer, ok := answerAry.([]interface{}); ok {
      answerString := answer[0].(string)
      answerTotals[answerString] += 1
      totalAnswers += 1
    } else {
      answerTotals["fail"] += 1
    }
  }
  answerTotals["totalAnswers"] = totalAnswers
  return answerTotals
}

type AnswersTotalHandler struct{}
func (a AnswersTotalHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  vars := mux.Vars(request)
  answers, err := GetAnswers(vars["id"])
  if err != nil {
    fmt.Fprintf(writer, err.Error())
  }
  totals := GetAnswerTotals(answers)
  WriteJSON(writer, totals)
}

func GetAnswerPercents(answerMap map[string]int) map[string]float64 {
  totalAnswers := float64(answerMap["totalAnswers"])
  answerPercentages := make(map[string]float64)
  for key, value := range answerMap {
    answerPercentages[key] = float64(value) / float64(totalAnswers)
  }
  return answerPercentages
}

type AnswersPercentHandler struct{}
func (a AnswersPercentHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
  vars := mux.Vars(request)
  answers, err := GetAnswers(vars["id"])
  if err != nil {
    fmt.Fprintf(writer, err.Error())
  }
  answerTotals := GetAnswerTotals(answers)
  answerPercents := GetAnswerPercents(answerTotals)
  WriteJSON(writer, answerPercents)
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
  answersTotalHandler := AnswersTotalHandler{}
  answersPercentHandler := AnswersPercentHandler{}
  router := mux.NewRouter()
  router.HandleFunc("/questions/new", NewQuestionHandler) // consider Filesystem handler so that css etc is also served
  router.Handle("/questions", QuestionsHandler)
  router.Handle("/questions{extension}", QuestionsHandler)
  router.Handle("/questions/{id}/answers", AnswersHandler)
  router.Handle("/questions/{id}/answers{extension}", AnswersHandler) 
  router.Handle("/questions/{id}/answers/total", answersTotalHandler)
  router.Handle("/questions/{id}/answers/percent", answersPercentHandler)
  http.Handle("/", router)
  http.ListenAndServe("localhost:4000", nil)
}