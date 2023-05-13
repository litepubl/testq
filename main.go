package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

//Приложению есть куда бесконечно расти
//обработка ошибок
// ограниченное количество сообщений
//вместо sync.Map использовать обычную мапу защищенную мютексом RW

type QList struct {
	channels sync.Map
}

func main() {
	q := QList{}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.Trim(r.URL.Path, "/")
		if name == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if r.Method == http.MethodPut {
			v := r.URL.Query().Get("v")
			if v == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			q.Store(name, v)
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodGet {
			ch := q.Load(name)
			t, err := strconv.Atoi(r.URL.Query().Get("timeout"))
			if err != nil {
				t = 0
			}

			if t == 0 {
				select {
				case v := <-ch:
					w.Write([]byte(v))

				default:
					w.WriteHeader(http.StatusNotFound)
				}

				return
			}

			timer := time.NewTimer(time.Duration(t) * time.Second)
			select {
			case v := <-ch:
				w.Write([]byte(v))
				timer.Stop()

			case <-timer.C:
				w.WriteHeader(http.StatusNotFound)
			}

			return
		}

		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	})

	port := "8080"
	if len(os.Args) > 1 && os.Args[1] != "" {
		port = os.Args[1]
	}

	err := http.ListenAndServe(":"+port, mux)
	if err != nil && err != http.ErrServerClosed {
		log.Printf("Error listen http server %v", err)
	}
}

func (q *QList) Store(name, value string) {
	var ch chan string
	c, ok := q.channels.Load(name)
	if !ok {
		ch = q.addChannel(name)
	} else {
		ch = c.(chan string)
	}

	ch <- value
}

func (q *QList) Load(name string) <-chan string {
	ch, ok := q.channels.Load(name)
	if ok {
		return ch.(chan string)
	}

	return q.addChannel(name)
}

func (q *QList) addChannel(name string) chan string {
	ch := make(chan string, 2048)
	q.channels.Store(name, ch)
	return ch
}
