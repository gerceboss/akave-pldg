package utils 

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)


// We are performing manual rpc calls instead of using the package so as to efficiently handle eth_getLogs block range errors 
//basically we will map error range to messages to decode the error and if the error was rate limit exceed we retry with full range but if the error was
// block range too large we will split the range and retry with smaller ranges until we get a successful response or we hit the minimum range threshold
// please do not change the logic of this function without understanding the above point as it is crucial for the performance of the indexer and also to avoid hitting rate limits frequently

func (r *RpcUrl) MakeRequest(method string, body []byte) (*http.Response, error) {
	
}


