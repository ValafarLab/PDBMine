package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	//Redis
	"github.com/gomodule/redigo/redis"
)

const (
	//RedisHost is the name of the redis server connection
	RedisHost string = "localhost"
	//RedisPort is the port number redis is listening on
	RedisPort int = 6379
	//TTL is the time to live in Redis
	//TTL string = "259200" //3 days
	//TTL string = "3600" //1 hour
	TTL string = "86400" //1 day
)

//ConnectToRedis attempt to make a connection to a redis cluster
func (db *ProteinDB) ConnectToRedis() error {
	//Get the host for Redis from the environment variable or default
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = RedisHost
		log.Printf("REDIS_HOST is not set. Defaulting to %s", redisHost)
	}

	//Get the port number for Redis from the environment variable or default
	rPort := os.Getenv("REDIS_PORT")
	rport, err := strconv.Atoi(rPort)
	if err != nil {
		rport = RedisPort
		log.Printf("REDIS_PORT is not set or is not valid. Defaulting to %d", rport)
	}

	//Try to connect to Redis and store the connection
	connString := fmt.Sprintf("%s:%d", redisHost, rport)
	log.Printf("Attempting to connect to Redis at %s\n", connString)
	db.Redis, err = redis.Dial("tcp", connString)

	return err
}

//RedisAvailable returns true if there's a good connection to Redis and false if not.
func (db *ProteinDB) RedisAvailable() bool {
	//If there's no connection, then don't try to save to redis
	if db.Redis == nil {
		log.Println("Redis connection unavailable.")
		return false
	}

	_, err := db.Redis.Do("PING")
	if err != nil {
		log.Println("Cannot Connect to Redis", err)
		log.Println("Retrying")
		db.ConnectToRedis()
		_, err = db.Redis.Do("PING")
		if err != nil {
			log.Println("Retry failed")
			return false
		}
	}

	log.Println("Connection Successful")
	return true
}

//GetQueryFromRedis checks to see if a query has already been saved. If it has it returns the queryID. If not it returns ""
func (db *ProteinDB) GetQueryFromRedis(post QueryRequest) (string, error) {
	queryKey, err := json.Marshal(post)
	if err != nil {
		return "", err
	}

	exists, _ := redis.Bool(db.Redis.Do("Exists", queryKey))

	if exists {
		queryString, err := redis.String(db.Redis.Do("HGET", queryKey, "queryID"))
		if err != nil {
			return "", err
		}

		return queryString, nil
	}

	return "", nil
}

//SaveQueryInRedis saves the query and the correstponsing queryID in redis so we don't duplicate work
func (db *ProteinDB) SaveQueryInRedis(post QueryRequest, queryID string) error {
	queryKey, err := json.Marshal(post)

	if err != nil {
		return err
	}

	_, err = db.Redis.Do("HMSET", queryKey, "queryID", queryID)
	if err != nil {
		return err
	}

	_, err = db.Redis.Do("EXPIRE", queryKey, TTL)
	if err != nil {
		return err
	}

	return nil
}

//GetResponseFromRedis retrieves a QueryResponse from Redis based on the query ID
func (db *ProteinDB) GetResponseFromRedis(queryID string) (QueryResponse, error) {
	log.Println("Getting status from Redis")
	resp := new(QueryResponse)

	exists, _ := redis.Bool(db.Redis.Do("Exists", queryID))

	if exists {
		status, err := redis.String(db.Redis.Do("HGET", queryID, "status"))
		if err != nil {
			return *resp, err
		}

		qs, err := ParseQueryStatus(status)
		log.Printf("Query status is %s\n", qs)
		if err != nil {
			return *resp, err
		}

		switch qs {
		case Error:
			resp.Status = status
			resp.QueryID = queryID
		case Running:
			resp.Status = status
			resp.QueryID = queryID
		case Complete:
			respString, err := redis.Bytes(db.Redis.Do("HGET", queryID, "result"))

			err = json.Unmarshal(respString, resp)
			resp.QueryID = queryID
			if err != nil {
				return *resp, err
			}
		default:
			return *resp, errors.New("unknown query status")
		}
	}

	return *resp, nil
}

//GetStatusFromRedis retrieves a QueryResponse from Redis based on the query ID
func (db *ProteinDB) GetStatusFromRedis(queryID string) (QueryResponse, error) {
	log.Println("Getting query status from Redis")
	resp := new(QueryResponse)

	exists, _ := redis.Bool(db.Redis.Do("Exists", queryID))

	if exists {
		status, err := redis.String(db.Redis.Do("HGET", queryID, "status"))
		if err != nil {
			return *resp, err
		}

		qs, err := ParseQueryStatus(status)
		log.Printf("Query status is %s\n", qs)
		if err != nil {
			return *resp, err
		}

		switch qs {
		case Error:
			resp.Status = status
			resp.QueryID = queryID
		case Running:
			resp.Status = status
			resp.QueryID = queryID
		case Complete:
			//respString, err := redis.Bytes(db.Redis.Do("HGET", queryID, "result"))

			//err = json.Unmarshal(respString, resp)
			resp.Status = status
			resp.QueryID = queryID
			//if err != nil {
			//	return *resp, err
			//}
		default:
			return *resp, errors.New("unknown query status")
		}
	}

	return *resp, nil
}

//UpdateStatusInRedis updates the status of a query in Redis based off query ID
func (db *ProteinDB) UpdateStatusInRedis(queryID string, status QueryStatus) error {
	_, err := db.Redis.Do("HMSET", queryID, "status", status)
	if err != nil {
		return err
	}

	_, err = db.Redis.Do("EXPIRE", queryID, TTL)
	if err != nil {
		return err
	}

	return nil
}

//SaveToRedis will be replaced by SaveFrameToRedis at some point
func (db *ProteinDB) SaveToRedis(name string, json []byte) error {
	_, err := db.Redis.Do("HMSET", name, "result", json)
	if err != nil {
		_, err = db.Redis.Do("HMSET", name, "status", Error)

	}

	_, err = db.Redis.Do("HMSET", name, "status", Complete)
	_, err = db.Redis.Do("EXPIRE", name, TTL)

	return err
}
