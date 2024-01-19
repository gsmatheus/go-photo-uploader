package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	baseURL      = "localhost"
	accessToken  = "Bearer token"
	uploadPath   = "/admin/gallery/add"
	productPath  = "/admin/products"
	addImagePath = "/admin/add-image"
)

func main() {
	client := &http.Client{}

	// Obter produtos
	productRequest, err := http.NewRequest("GET", baseURL+productPath, nil)
	if err != nil {
		fmt.Printf("Erro ao criar a solicitação de produtos: %v\n", err)
		return
	}

	productRequest.Header.Set("Content-Type", "application/json")
	productRequest.Header.Set("Authorization", accessToken)

	productResponse, err := client.Do(productRequest)
	if err != nil {
		fmt.Printf("Erro ao obter produtos: %v\n", err)
		return
	}
	defer productResponse.Body.Close()

	// Decodificar a resposta JSON
	var products map[string]interface{}
	if err := json.NewDecoder(productResponse.Body).Decode(&products); err != nil {
		fmt.Printf("Erro ao decodificar a resposta JSON: %v\n", err)
		return
	}

	// Processar produtos e fotos
	fotos := make(map[string][]string)

	for _, produto := range products["data"].([]interface{}) {
		reference := produto.(map[string]interface{})["reference"].(string)
		id := int(produto.(map[string]interface{})["id"].(float64))

		// Evitar algumas referências específicas
		if reference == "1100" || reference == "1102" || reference == "1103" {
			continue
		}

		fotos[fmt.Sprintf("%s_%d", reference, id)] = []string{}
	}

	arquivos, err := pegaFotos("fotos")
	if err != nil {
		fmt.Printf("Erro ao obter a lista de arquivos: %v\n", err)
		return
	}

	for _, arquivo := range arquivos {
		for referencia := range fotos {
			if strings.Contains(arquivo, "-") {
				ids := strings.Split(strings.Split(arquivo, ".")[0], "-")
				for _, id := range ids {
					if id == strings.Split(referencia, "_")[0] {
						fmt.Printf("Encontrado %s para %s\n", arquivo, referencia)
						fotos[referencia] = append(fotos[referencia], arquivo)
					}
				}
			} else {
				if strings.Split(referencia, "_")[0] == strings.Split(arquivo, ".")[0] {
					fmt.Printf("Encontrado %s para %s\n", arquivo, referencia)
					fotos[referencia] = append(fotos[referencia], arquivo)
				}
			}
		}
	}

	for referencia, produtoID := range fotos {
		arquivos := produtoID
		fmt.Printf("%s %s %v\n", referencia, produtoID, arquivos)

		for _, arquivo := range arquivos {
			galeria, err := uploadFoto(arquivo)
			if err != nil {
				fmt.Printf("Erro ao enviar foto %s: %v\n", arquivo, err)
				continue
			}

			vinculaFoto(galeria.Gallery.Props.ID, referencia)
		}
	}
}

func pegaFotos(caminhoPasta string) ([]string, error) {
	arquivos, err := os.ReadDir(caminhoPasta)
	if err != nil {
		return nil, fmt.Errorf("Erro ao listar arquivos: %v", err)
	}

	var result []string
	for _, arquivo := range arquivos {
		if arquivo.IsDir() {
			continue
		}
		result = append(result, arquivo.Name())
	}

	return result, nil
}

func uploadFoto(foto string) (*Galeria, error) {
	caminhoCompleto := filepath.Join("fotos", foto)

	// Abrir a imagem usando a biblioteca padrão de imagens do Go
	img, _, err := image.DecodeFile(caminhoCompleto)
	if err != nil {
		return nil, fmt.Errorf("Erro ao abrir a imagem: %v", err)
	}

	// Criar um buffer de bytes para armazenar a imagem comprimida
	imgBuffer := new(bytes.Buffer)
	err = jpeg.Encode(imgBuffer, img, &jpeg.Options{Quality: 85})
	if err != nil {
		return nil, fmt.Errorf("Erro ao comprimir a imagem: %v", err)
	}

	// Criar o objeto MultipartWriter
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Adicionar a imagem ao formulário multipart
	part, err := writer.CreateFormFile("file", foto)
	if err != nil {
		return nil, fmt.Errorf("Erro ao criar o formulário multipart: %v", err)
	}
	part.Write(imgBuffer.Bytes())

	// Finalizar o formulário multipart
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("Erro ao finalizar o formulário multipart: %v", err)
	}

	// Criar a solicitação POST
	req, err := http.NewRequest("POST", baseURL+uploadPath, body)
	if err != nil {
		return nil, fmt.Errorf("Erro ao criar a solicitação POST: %v", err)
	}

	req.Header.Set("Authorization", accessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Enviar a solicitação
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Erro ao enviar a solicitação: %v", err)
	}
	defer resp.Body.Close()

	// Decodificar a resposta JSON
	var galeria Galeria
	if err := json.NewDecoder(resp.Body).Decode(&galeria); err != nil {
		return nil, fmt.Errorf("Erro ao decodificar a resposta JSON: %v", err)
	}

	return &galeria, nil
}

func vinculaFoto(galeriaID int, referencia string) {
	url := fmt.Sprintf("%s/%s", baseURL, addImagePath)

	data := map[string]interface{}{
		"gallery_ids": []int{galeriaID},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Erro ao converter dados para JSON: %v\n", err)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Erro ao criar a solicitação POST: %v\n", err)
		return
	}

	req.Header.Set("Authorization", accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Erro ao enviar a solicitação: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("Foto vinculada com sucesso! %d %s\n", galeriaID, referencia)
	} else {
		fmt.Printf("Erro ao vincular foto: %d %s\n", resp.StatusCode, referencia)
	}
}

// Galeria representa a estrutura da resposta JSON da criação de uma galeria
type Galeria struct {
	Gallery struct {
		Props struct {
			ID int `json:"id"`
		} `json:"props"`
	} `json:"gallery"`
}
