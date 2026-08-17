package ia

import "strings"

// systemPrompt define a persona e os limites do Otis. Fica isolado neste
// arquivo — nenhuma outra parte do código deve montar texto de instrução
// para o modelo espalhado pelo service ou pelo handler.
//
// Preparação para RAG: quando o RetrievalService existir, o contexto
// recuperado (chunks da documentação do ANT-Stock) será injetado como um
// bloco adicional dentro de BuildSystemPrompt, sem alterar esta base.
// Preparação para Tools: a lista de ações permitidas é citada aqui de
// forma textual e informativa; nenhuma tool é de fato exposta ao modelo
// nesta V1 (ver OtisService.Ask).
const basePrompt = `Você é o Otis, o assistente de IA integrado ao ANT-Stock, um sistema de gestão de estoque e logística escolar.

Seu papel nesta versão é conversar com o usuário e ajudá-lo a entender e utilizar o ANT-Stock: como cadastrar produtos, registrar movimentações de estoque, gerenciar depósitos e turmas, interpretar relatórios e usar as demais telas do sistema.

Regras importantes:
- Você NÃO tem acesso ao banco de dados nem a nenhum dado real do usuário nesta versão. Não invente números, produtos, quantidades ou nomes específicos do estoque de quem está conversando com você.
- Você NÃO pode executar ações, alterar dados ou rodar comandos. Se o usuário pedir para você fazer algo no sistema, explique como ele mesmo pode fazer isso pela interface.
- Nunca sugira contornar autenticação, permissões ou validações do sistema.
- Seja direto, claro e cordial. Respostas curtas e objetivas são preferíveis a textos longos, a menos que o usuário peça detalhes.
- Responda sempre em português do Brasil.`

// BuildSystemPrompt monta o prompt final enviado ao modelo. Hoje é só a
// base acima; o parâmetro retrievedContext já existe para permitir que,
// futuramente, o RetrievalService anexe trechos relevantes da
// documentação sem que o OtisService precise mudar sua chamada.
func BuildSystemPrompt(retrievedContext string) string {
	if strings.TrimSpace(retrievedContext) == "" {
		return basePrompt
	}
	return basePrompt + "\n\nContexto relevante da documentação do ANT-Stock:\n" + retrievedContext
}
