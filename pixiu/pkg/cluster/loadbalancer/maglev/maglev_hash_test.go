/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package maglev

import (
	"fmt"
	stdHttp "net/http"
	"strconv"
	"testing"
)

import (
	"github.com/apache/dubbo-go-pixiu/pixiu/pkg/context/http"
	"github.com/apache/dubbo-go-pixiu/pixiu/pkg/model"
)

func TestMaglevHash(t *testing.T) {

	nodeCount := 5

	nodes := make([]*model.Endpoint, 0, nodeCount)

	for i := 1; i <= nodeCount; i++ {
		name := strconv.Itoa(i)
		nodes = append(nodes, &model.Endpoint{ID: name, Name: name,
			Address: model.SocketAddress{Address: "192.168.1." + name, Port: 1000 + i}})
	}

	cluster := &model.ClusterConfig{
		Name:           "test-cluster",
		Endpoints:      nodes,
		LbStr:          model.LoadBalancerMaglevHashing,
		ConsistentHash: model.ConsistentHash{MaglevTableSize: 521},
	}
	cluster.CreateConsistentHash()

	var (
		hashing = MaglevHash{}
		path    string
	)

	for i := 1; i <= 20; i++ {
		path = fmt.Sprintf("/pixiu?total=%d", i)
		t.Log(hashing.Handler(cluster, &http.HttpContext{Request: &stdHttp.Request{Method: stdHttp.MethodGet, RequestURI: path}}))
	}

}
