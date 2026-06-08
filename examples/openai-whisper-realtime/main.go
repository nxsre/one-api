// OpenAI 实时语音转文本示例（Realtime Transcription API + gpt-realtime-whisper）
//
// whisper-1 走 /v1/audio/transcriptions，适合整段文件，不支持 WebSocket 增量输出。
// 实时转写请用 Realtime API，通过 WebSocket 推送 PCM 音频并接收 transcript delta。
//
// 用法:
//
//	# 1) 从 WAV 文件模拟实时推流（需 24kHz / 16-bit / mono）
//	export OPENAI_API_KEY=sk-...
//	go run . -file sample.wav
//
//	# 2) 从 stdin 读取原始 PCM（配合 ffmpeg 采集麦克风）
//	ffmpeg -f avfoundation -i ":0" -ar 24000 -ac 1 -f s16le - 2>/dev/null | go run . -stdin
//
//	# 3) 指定语言与延迟档位（minimal/low/medium/high/xhigh）
//	go run . -file sample.wav -lang zh -delay low
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

const (
	realtimeURL = "wss://api.openai.com/v1/realtime?model=gpt-realtime-whisper"
	sampleRate  = 24000
	bytesPerMs  = sampleRate * 2 / 1000 // PCM16 mono: 2 bytes per sample
)

func main() {
	filePath := flag.String("file", "", "WAV 文件路径（24kHz, 16-bit, mono）")
	useStdin := flag.Bool("stdin", false, "从 stdin 读取原始 PCM16 数据")
	lang := flag.String("lang", "zh", "语言提示，如 zh / en")
	delay := flag.String("delay", "low", "延迟档位: minimal, low, medium, high, xhigh")
	chunkMs := flag.Int("chunk-ms", 100, "每次 append 的音频块时长（毫秒）")
	commitMs := flag.Int("commit-ms", 800, "每隔多久 commit 一次缓冲区（毫秒）")
	flag.Parse()

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		log.Fatal("请设置环境变量 OPENAI_API_KEY")
	}
	if (*filePath == "" && !*useStdin) || (*filePath != "" && *useStdin) {
		log.Fatal("请指定 -file 或 -stdin 之一")
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer "+apiKey)
	header.Set("OpenAI-Beta", "realtime=v1")

	conn, _, err := websocket.DefaultDialer.Dial(realtimeURL, header)
	if err != nil {
		log.Fatalf("连接 WebSocket 失败: %v", err)
	}
	defer conn.Close()

	done := make(chan struct{})
	go readEvents(conn, done)

	if err := sendSessionUpdate(conn, *lang, *delay); err != nil {
		log.Fatalf("发送 session.update 失败: %v", err)
	}

	// 等待 session 就绪
	time.Sleep(500 * time.Millisecond)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	streamDone := make(chan error, 1)
	go func() {
		var r io.Reader
		if *useStdin {
			r = os.Stdin
		} else {
			pcm, err := loadWAVPCM(*filePath)
			if err != nil {
				streamDone <- err
				return
			}
			r = bytes.NewReader(pcm)
		}
		streamDone <- streamAudio(conn, r, *chunkMs, *commitMs)
	}()

	select {
	case err := <-streamDone:
		if err != nil {
			log.Fatalf("推流失败: %v", err)
		}
	case <-interrupt:
		log.Println("\n收到中断信号，正在退出...")
	}

	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	close(done)
	time.Sleep(300 * time.Millisecond)
}

func sendSessionUpdate(conn *websocket.Conn, lang, delay string) error {
	payload := map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"type": "transcription",
			"audio": map[string]any{
				"input": map[string]any{
					"format": map[string]any{
						"type": "audio/pcm",
						"rate": sampleRate,
					},
					"transcription": map[string]any{
						"model":    "gpt-realtime-whisper",
						"language": lang,
						"delay":    delay,
					},
					"turn_detection": nil,
				},
			},
		},
	}
	return conn.WriteJSON(payload)
}

func streamAudio(conn *websocket.Conn, r io.Reader, chunkMs, commitMs int) error {
	chunkSize := bytesPerMs * chunkMs
	buf := make([]byte, chunkSize)
	lastCommit := time.Now()

	for {
		n, err := io.ReadFull(r, buf)
		if err == io.ErrUnexpectedEOF {
			if n > 0 {
				if err := appendAudio(conn, buf[:n]); err != nil {
					return err
				}
			}
			break
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := appendAudio(conn, buf[:n]); err != nil {
			return err
		}

		if time.Since(lastCommit) >= time.Duration(commitMs)*time.Millisecond {
			if err := commitBuffer(conn); err != nil {
				return err
			}
			lastCommit = time.Now()
		}

		// 模拟实时采集节奏
		time.Sleep(time.Duration(chunkMs) * time.Millisecond)
	}

	// 收尾：提交剩余音频
	if err := commitBuffer(conn); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
}

func appendAudio(conn *websocket.Conn, pcm []byte) error {
	return conn.WriteJSON(map[string]any{
		"type":  "input_audio_buffer.append",
		"audio": base64.StdEncoding.EncodeToString(pcm),
	})
}

func commitBuffer(conn *websocket.Conn) error {
	return conn.WriteJSON(map[string]any{
		"type": "input_audio_buffer.commit",
	})
}

func readEvents(conn *websocket.Conn, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			log.Printf("读取消息失败: %v", err)
			return
		}

		var event map[string]json.RawMessage
		if err := json.Unmarshal(msg, &event); err != nil {
			log.Printf("解析事件失败: %v", err)
			continue
		}

		var eventType string
		if err := json.Unmarshal(event["type"], &eventType); err != nil {
			continue
		}

		switch eventType {
		case "conversation.item.input_audio_transcription.delta":
			var delta struct {
				Delta string `json:"delta"`
			}
			if err := json.Unmarshal(msg, &delta); err == nil && delta.Delta != "" {
				fmt.Print(delta.Delta)
			}
		case "conversation.item.input_audio_transcription.completed":
			var completed struct {
				Transcript string `json:"transcript"`
				ItemID     string `json:"item_id"`
			}
			if err := json.Unmarshal(msg, &completed); err == nil {
				fmt.Printf("\n[完成 %s] %s\n", completed.ItemID, completed.Transcript)
			}
		case "error":
			log.Printf("服务端错误: %s", string(msg))
		case "session.created", "session.updated":
			log.Printf("会话就绪: %s", eventType)
		}
	}
}

// loadWAVPCM 读取标准 PCM WAV，要求 16-bit mono 24kHz。
func loadWAVPCM(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 44 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, fmt.Errorf("不是有效的 WAV 文件")
	}

	offset := 12
	var sampleRateFound uint32
	var bitsPerSample, numChannels uint16
	var dataOffset, dataSize int

	for offset+8 <= len(data) {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		chunkStart := offset + 8

		switch chunkID {
		case "fmt ":
			if chunkSize >= 16 {
				numChannels = binary.LittleEndian.Uint16(data[chunkStart+2 : chunkStart+4])
				sampleRateFound = binary.LittleEndian.Uint32(data[chunkStart+4 : chunkStart+8])
				bitsPerSample = binary.LittleEndian.Uint16(data[chunkStart+14 : chunkStart+16])
			}
		case "data":
			dataOffset = chunkStart
			dataSize = chunkSize
		}

		offset = chunkStart + chunkSize
		if chunkSize%2 == 1 {
			offset++
		}
	}

	if dataOffset == 0 || dataSize == 0 {
		return nil, fmt.Errorf("WAV 缺少 data chunk")
	}
	if numChannels != 1 {
		return nil, fmt.Errorf("需要 mono 音频，当前 channels=%d", numChannels)
	}
	if bitsPerSample != 16 {
		return nil, fmt.Errorf("需要 16-bit PCM，当前 bits=%d", bitsPerSample)
	}
	if int(sampleRateFound) != sampleRate {
		return nil, fmt.Errorf("需要 %d Hz，当前 %d Hz（可用 ffmpeg 转换: ffmpeg -i in.wav -ar 24000 -ac 1 out.wav）", sampleRate, sampleRateFound)
	}

	end := dataOffset + dataSize
	if end > len(data) {
		end = len(data)
	}
	pcm := make([]byte, end-dataOffset)
	copy(pcm, data[dataOffset:end])
	return pcm, nil
}
