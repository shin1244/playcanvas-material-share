package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/playwright-community/playwright-go"
)

// Tampermonkey에서 보낸 JSON 요청을 받을 구조체
type SyncRequest struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

func main() {
	file, err := os.Open("./url.txt") // 같은 디렉토리에 있다고 가정
	if err != nil {
		log.Fatalf("Failed to open URL file: %v", err)
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text()) // 공백 제거
		if line == "" {
			continue // 빈 줄은 건너뜀
		}
		urls = append(urls, line)
	}

	page, err := playcanvasLogin()
	if err != nil {
		log.Fatalf("Playcanvas login failed: %v", err)
	}

	r := gin.Default()

	// --- CORS 설정 (동일) ---
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"https://playcanvas.com"}
	config.AllowMethods = []string{"POST", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type"}
	r.Use(cors.New(config))

	// --- POST 핸들러 ---
	r.POST("/sync-material", func(c *gin.Context) {
		log.Println("Received /sync-material request")
		var req []SyncRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body: " + err.Error()})
			return
		}

		for _, targetSceneURL := range urls {
			page.Goto(targetSceneURL)
			if err != nil {
				log.Printf("Failed to load page %s: %v", targetSceneURL, err)
				continue
			}
			syncMaterial(req, page)
			fmt.Println("--------------------------------")
			log.Printf("Finished syncing materials for scene: %s", targetSceneURL)
			fmt.Println("--------------------------------")
		}

		// Tampermonkey 스크립트에는 즉시 성공 응답을 보냅니다.
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "데이터 수신 완료. 서버에서 자동화 작업을 시작합니다.",
		})
	})

	fmt.Println()
	log.Println("http://localhost:8080/sync-material")
	fmt.Println()

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Gin 서버 실행 실패: %v", err)
	}
}

func playcanvasLogin() (playwright.Page, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, err
	}

	authFile := "auth.json"
	headless := true // 기본값: 로그인 되어있으면 헤드리스

	// 먼저 auth.json 존재 여부 확인
	if _, err := os.Stat(authFile); os.IsNotExist(err) {
		headless = false // 로그인 필요 시 헤드리스 끄기
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
	})
	if err != nil {
		return nil, err
	}

	var context playwright.BrowserContext

	if _, err := os.Stat(authFile); err == nil {
		// --- auth.json 존재, 저장된 세션 사용 ---
		context, err = browser.NewContext(playwright.BrowserNewContextOptions{
			StorageStatePath: playwright.String(authFile),
		})
		if err != nil {
			return nil, err
		}
		log.Println("Playcanvas: Logged in using existing auth.json")

		page, err := context.NewPage()
		if err != nil {
			return nil, err
		}

		if _, err := page.Goto("https://playcanvas.com/"); err != nil {
			return nil, err
		}

		err = page.WaitForURL("https://playcanvas.com/user/*", playwright.PageWaitForURLOptions{
			Timeout: playwright.Float(10000),
		})
		if err != nil {
			log.Println("auth.json 사용했으나 유저 페이지 이동 실패:", err)
			return nil, err
		}
		log.Println("✅ auth.json으로 로그인 완료, 유저 페이지 도착")
		return page, nil

	} else {
		// --- auth.json 없으면 최초 로그인 ---
		context, err = browser.NewContext()
		if err != nil {
			return nil, err
		}
		page, err := context.NewPage()
		if err != nil {
			return nil, err
		}

		log.Println("Playcanvas: 최초 로그인 시도...")
		if _, err := page.Goto("https://login.playcanvas.com/"); err != nil {
			return nil, err
		}

		// 로그인 완료 대기
		err = page.WaitForURL("https://playcanvas.com/user/*", playwright.PageWaitForURLOptions{
			Timeout: playwright.Float(0),
		})
		if err != nil {
			log.Println("로그인 후 URL 이동 실패:", err)
			return nil, err
		}
		log.Println("✅ 최초 로그인 완료, 유저 페이지 도착")

		// 로그인 후 auth.json 저장
		state, err := context.StorageState()
		if err != nil {
			return nil, err
		}
		jsonData, err := json.Marshal(state)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(authFile, jsonData, 0644); err != nil {
			return nil, err
		}
		log.Println("Playcanvas: Logged in and saved auth.json")

		return page, nil
	}
}

func syncMaterial(data []SyncRequest, page playwright.Page) {
	waitScript := `() => {
		return new Promise((resolve, reject) => {
			const checkAssets = () => {
				if (typeof editor !== 'undefined' && editor && typeof editor.call === 'function') {
					allAssets = editor.call('assets:list');
					if (allAssets && allAssets.length > 0) {
						console.log('Assets list populated. Count:', allAssets.length);
						resolve(true);
					} else {
						console.log('Editor ready, assets list is empty. Retrying in 1000ms...');
						setTimeout(checkAssets, 1000);
					}
				} else {
					console.log('Editor API not ready. Retrying in 1000ms...');
					setTimeout(checkAssets, 1000);
				}
			};
			checkAssets();
		});
	}`

	_, err := page.Evaluate(waitScript)

	if err != nil {
		log.Fatalf("Failed to wait for assets: %v", err)
		return
	}

	for _, material := range data {
		materialData, ok := material.Data.(map[string]any)
		if !ok {
			log.Printf("Invalid material data format for %s", material.Name)
			continue
		}
		mapImages := findMapImage(materialData)
		imagesJson, _ := json.Marshal(mapImages)
		newData, reflectivity := replaceReflectivity(materialData)
		newDataJSON, err := json.Marshal(newData)
		if err != nil {
			log.Printf("Failed to marshal material data for %s: %v", material.Name, err)
			continue
		}
		script := fmt.Sprintf(`() => {
            let materialName = '%s';
            let newData = %s;
            let mapImages = %s;
            let reflectivity = %f;

            console.log('Syncing material: ' + materialName);
            let material = allAssets.find(asset => asset.get('name') === materialName && asset.get('type') === 'material');
            
            if (!material) {
                console.log('Material not found: ' + materialName);
                return false;
            }

            Object.entries(mapImages).forEach(([key, value]) => {
                let targetAsset = editor.call('assets:findOne', asset =>
                    asset && typeof asset.get === 'function' && asset.get('name') === value
                );
                if (targetAsset) {
                    newData[key] = targetAsset[1].get('id');
                    console.log(newData[key]);
                } else {
                    console.warn("Image asset not found for map:", key, "with name:", value);
                }
            });

            material.set('data', newData);
            material.set('data.reflectivity', reflectivity);
            
            return true;

        }`, material.Name, newDataJSON, imagesJson, reflectivity)

		result, err := page.Evaluate(script)
		if err != nil {
			log.Printf("Failed to sync material %s: %v", material.Name, err)
			continue
		}
		if success, ok := result.(bool); ok && success {
			log.Printf("Material synced: %s", material.Name)
		} else {
			log.Printf("Can not find Material: %s", material.Name)
		}
	}
}

func replaceReflectivity(data map[string]any) (map[string]any, float64) {
	newData := make(map[string]any, len(data))
	for k, v := range data {
		newData[k] = v
	}
	reflectivityValue := newData["reflectivity"].(float64)
	newData["reflectivity"] = 1.0

	return newData, reflectivityValue
}

func findMapImage(data map[string]any) map[string]string {
	var images = make(map[string]string)
	for key, value := range data {
		if strings.HasSuffix(key, "Map") {
			if imgName, ok := value.(string); ok && imgName != "" {
				images[key] = imgName
			}
		}
	}
	return images
}

// func replaceMapTiling(data map[string]any, tilingX, tilingY float64) map[string]any {
// 	keys := []string{
// 		"aoMapTiling",
// 		"clearCoatGlossMapTiling",
// 		"clearCoatMapTiling",
// 		"clearCoatNormalMapTiling",
// 		"diffuseMapTiling",
// 		"emissiveMapTiling",
// 		"glossMapTiling",
// 		"heightMapTiling",
// 		"iridescenceMapTiling",
// 		"iridescenceThicknessMapTiling",
// 		"lightMapTiling",
// 		"metalnessMapTiling",
// 		"normalMapTiling",
// 		"opacityMapTiling",
// 		"refractionMapTiling",
// 		"sheenGlossMapTiling",
// 		"sheenMapTiling",
// 		"specularMapTiling",
// 		"specularityFactorMapTiling",
// 		"thicknessMapTiling",
// 		"anisotropyMapTiling",
// 		"reflectivity",
// 	}

// 	for _, key := range keys {
// 		if key == "reflectivity" {
// 			delete(data, key)
// 			continue
// 		}
// 		data[key] = []float64{tilingX, tilingY}
// 	}
// 	return data
// }
