// Пакет реализует очередь сообщений.
// Предусмотренна возможность ожидания нового значения, если очередь пуста.
// Так же есть возможность указать порт первым аргументом строки.
package main

import (
	"container/list"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	defaultPort       = 9090
	putValueHTTPKey   = "v"
	getTimeoutHTTPKey = "timeout"
)

// QueuesManager конкурентно безопасен.
// Содержит в себе мапы с очередями данных и каналов для ожидания новых данных по таймауту.
// Менеджер не удаляет неиспользуемые очереди!
type QueuesManager struct {
	sync.Mutex
	queues map[string]*queueList
}

// queueList содержит в себе данные, либо каналы слушателей ожидающих новые данные.
type queueList struct {
	list.List
	// Флаг который определяет находятся ли в очереди слушатели ожидающие данных
	listeners bool
}

// NewQueuesManager возвращает инициализированную очередь.
func NewQueuesManager() *QueuesManager {
	list.New()
	return &QueuesManager{
		queues: make(map[string]*queueList),
	}
}

// Put кладет сообщение в очередь, либо сразу отдает ожидающим новых данных.
func (qm *QueuesManager) Put(qname, value string) {
	if qname == "" || value == "" {
		return
	}
	qm.Lock()
	defer qm.Unlock()
	queue := qm.queues[qname]
	if queue == nil {
		queue = &queueList{}
		queue.PushBack(value)
		qm.queues[qname] = queue
		return
	}

	if queue.listeners {
		e := queue.Front()
		e.Value.(chan string) <- value
		queue.Remove(e)
		if queue.Len() == 0 {
			queue.listeners = false
		}
		return
	}

	queue.PushBack(value)
}

// Get получает данные из очереди, если данных нет возращает пустую строку.
// Если timeout не nil, то ожидает до прихода сообщения, либо до истечения таймаута.
// Первым сообщение получит тот, кто первее запросил.
func (qm *QueuesManager) Get(qname string, timeout *time.Duration) string {
	if qname == "" {
		return ""
	}
	qm.Lock()
	defer func() {
		if qm.TryLock() {
			qm.Unlock()
		}
	}()
	queue := qm.queues[qname]
	if queue == nil {
		queue = &queueList{}
		qm.queues[qname] = queue
	}
	if timeout != nil && (queue.listeners || queue.Len() == 0) {
		queue.listeners = true // очередь пуста, значит в ней можно хранить каналы слушателей
		recieveChan := make(chan string)
		e := queue.PushBack(recieveChan)
		qm.Unlock()
		for {
			select {
			case v := <-recieveChan:
				return v
			case <-time.After(*timeout):
				qm.Lock()
				queue.Remove(e)
				if queue.Len() == 0 {
					queue.listeners = false
				}
				qm.Unlock()
				return ""
			}
		}
	}

	e := queue.Front()
	if e == nil {
		return ""
	}
	v := e.Value.(string)
	queue.Remove(e)

	return v
}

// Handler обрабатывает входящие запросы.
type Handler struct {
	qm *QueuesManager
}

// ServeHTTP обработчик запросов, принимает только GET и PUT.
// В пути указывается название очереди, куда нужно записать значение, либо получить.
// При PUT запросе должен присутсвовать параметр "v", которое является значением,
// которое нужно положить в очередь.
// При GET запросе можно установить параметр "timeout" в секундах, в течении которого
// приложение будет ожидать нового значения.
func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.Method == http.MethodPut {
		q := req.URL.Query()
		v := q.Get(putValueHTTPKey)
		if v == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		h.qm.Put(req.URL.Path, v)
		w.WriteHeader(http.StatusOK)
	} else if req.Method == http.MethodGet {
		q := req.URL.Query()
		v := q.Get(getTimeoutHTTPKey)
		timeout := new(time.Duration)
		if v != "" {
			t, err := strconv.Atoi(v)
			if err == nil {
				d := time.Second * time.Duration(t)
				timeout = &d
			}
		}

		res := h.qm.Get(req.URL.Path, timeout)
		if res == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		_, err := fmt.Fprint(w, res)
		if err != nil {
			log.Panicln(err)
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func main() {
	port := defaultPort
	// Если нужно указать порт, то он вводится первым аргуметом
	if len(os.Args) == 2 {
		portArg := os.Args[1]
		var err error
		port, err = strconv.Atoi(portArg)
		if err != nil {
			log.Fatalf(
				"Введите порт на котором должено работать приложени первым аргументом. Ошибка: %v",
				err,
			)
		}
	}

	log.Println(
		http.ListenAndServe(fmt.Sprintf("127.0.0.1:%v", port),
			&Handler{qm: NewQueuesManager()}),
	)
}
