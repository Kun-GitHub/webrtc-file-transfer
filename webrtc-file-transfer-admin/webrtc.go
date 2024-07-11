package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/pion/webrtc/v3"
)

// peerConnection 结构体用于存储每个对等连接的信息
type peerConnection struct {
	pc         *webrtc.PeerConnection     // WebRTC对等连接
	sdpChannel chan<- *webrtc.SessionDescription // SDP（会话描述协议）的通道
	candidateC chan<- *webrtc.ICECandidate       // ICE候选的通道
}

// 主函数设置HTTP路由并启动HTTP服务器
func main() {
	http.HandleFunc("/offer", handleOffer)      // 处理offer请求
	http.HandleFunc("/answer", handleAnswer)    // 处理answer请求
	http.HandleFunc("/candidate", handleCandidate) // 处理ICE候选请求

	log.Println("监听端口 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil { // 启动HTTP服务器
		log.Fatal("启动服务器失败:", err)
	}
}

// handleOffer 处理来自客户端的offer请求
func handleOffer(w http.ResponseWriter, r *http.Request) {
	offerSDP := &webrtc.SessionDescription{} // 创建一个会话描述结构体
	if err := json.NewDecoder(r.Body).Decode(offerSDP); err != nil { // 解析请求体中的SDP
		http.Error(w, "解析SDP失败", http.StatusBadRequest)
		return
	}

	answerPC := createPeerConnection() // 创建一个新的对等连接实例
	go answerPC.run(w, r.Context())   // 在新的goroutine中运行对等连接

	// 将offer SDP发送给answerPC
	answerPC.sdpChannel <- offerSDP
}

// handleAnswer 处理来自客户端的answer请求
func handleAnswer(w http.ResponseWriter, r *http.Request) {
	answerSDP := &webrtc.SessionDescription{}
	if err := json.NewDecoder(r.Body).Decode(answerSDP); err != nil {
		http.Error(w, "解析SDP失败", http.StatusBadRequest)
		return
	}

	offerPC := createPeerConnection()
	go offerPC.run(w, r.Context())

	// 将answer SDP发送给offerPC
	offerPC.sdpChannel <- answerSDP
}

// handleCandidate 处理来自客户端的ICE候选请求
func handleCandidate(w http.ResponseWriter, r *http.Request) {
	candidate := &webrtc.ICECandidate{}
	if err := json.NewDecoder(r.Body).Decode(candidate); err != nil {
		http.Error(w, "解析ICE候选失败", http.StatusBadRequest)
		return
	}

	// 假设只有一个连接，为简化起见
	pc := createPeerConnection()
	go func() {
		// 将ICE候选发送给peerConnection
		select {
		case pc.candidateC <- candidate:
		case <-r.Context().Done(): // 如果请求上下文已完成，则退出
			return
		}
	}()
}

// createPeerConnection 创建一个新的peerConnection实例
func createPeerConnection() *peerConnection {
	// 配置WebRTC连接
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{
			URLs: []string{"stun:stun.l.google.com:19302"}, // 使用Google的STUN服务器
		}},
	}
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("创建PeerConnection失败: %v", err)
	}

	// 创建SDP和ICE候选的channel
	sdpChannel := make(chan *webrtc.SessionDescription, 1)
	candidateC := make(chan *webrtc.ICECandidate, 1)

	return &peerConnection{
		pc:         pc,
		sdpChannel: sdpChannel,
		candidateC: candidateC,
	}
}

// peerConnection的run方法，处理WebRTC连接的生命周期
func (pc *peerConnection) run(w http.ResponseWriter, ctx context.Context) {
	// 在这里添加音视频轨道和设置事件处理器
	// ...

	// 设置远程描述（SDP）
	if _, ok := <-pc.sdpChannel; ok {
		if err := pc.pc.SetRemoteDescription(*<-pc.sdpChannel); err != nil {
			http.Error(w, "设置远程描述失败", http.StatusInternalServerError)
			return
		}
	}

	// 创建offer或answer，取决于你是发起方还是应答方
	offer, err := pc.pc.CreateOffer(nil)
	if err != nil {
		http.Error(w, "创建offer失败", http.StatusInternalServerError)
		return
	}
	if err := pc.pc.SetLocalDescription(offer); err != nil {
		http.Error(w, "设置本地描述失败", http.StatusInternalServerError)
		return
	}

	// 将offer或answer通过信令服务器发送回客户端
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(offer)

	// 处理ICE候选
	for {
		select {
		case candidate := <-pc.candidateC:
			// 将ICE候选通过信令服务器发送回客户端
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(candidate)
		case <-ctx.Done():
			return // 如果上下文已完成，退出goroutine
		}
	}
}
