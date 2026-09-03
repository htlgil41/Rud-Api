package libs

import (
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

type JoseManagerToken struct {
	Signer            jose.Signer
	TokenSecretAccess string
	ClaisnAccess      jwt.Claims
}

type PayloadCustom struct {
	IdUser   string `json:"id"`
	Username string `json:"username"`
}

func (j *JoseManagerToken) GenerateAccessTokens(
	idToken string,
	custom PayloadCustom,
) (string, error) {
	tokenStrinng := jwt.Signed(j.Signer)
	token, err := tokenStrinng.Claims(j.ClaisnAccess).Claims(custom).Serialize()
	if err != nil {
		return "", err
	}
	return token, nil
}

func (j *JoseManagerToken) ValidateAccessToken(
	token string,
) (PayloadCustom, error) {
	parseSigne, errParseSigne := jwt.ParseSigned(
		token,
		[]jose.SignatureAlgorithm{jose.ES256},
	)
	if errParseSigne != nil {
		return PayloadCustom{}, errParseSigne
	}

	var clains jwt.Claims
	var customClain PayloadCustom

	if errParse := parseSigne.Claims(j.TokenSecretAccess, &clains, &customClain); errParse != nil {
		return PayloadCustom{}, errParse
	}
	return customClain, nil
}
