package user

import (
	"diskette/collections"
	"diskette/tokens"
	"diskette/util"
	"errors"
	"net/http"
	"time"

	"github.com/satori/go.uuid"

	"github.com/labstack/echo"
	"golang.org/x/crypto/bcrypt"
	"labix.org/v2/mgo/bson"
)

// http POST localhost:5025/user/signup name="Joe Doe" email=joe.doe@gmail.com password=abc language=en
func (service *serviceImpl) Signup(c *echo.Context) error {
	var request struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Language string `json:"language"`
	}
	c.Bind(&request)

	if request.Name == "" {
		return c.JSON(http.StatusBadRequest, util.CreateErrResponse(errors.New("Missing parameter 'name'")))
	}

	if request.Email == "" {
		return c.JSON(http.StatusBadRequest, util.CreateErrResponse(errors.New("Missing parameter 'email'")))
	}

	if request.Password == "" {
		return c.JSON(http.StatusBadRequest, util.CreateErrResponse(errors.New("Missing parameter 'password'")))
	}

	if request.Language == "" {
		return c.JSON(http.StatusBadRequest, util.CreateErrResponse(errors.New("Missing parameter 'language'")))
	}

	return service.createUser(c, request.Name, request.Email, request.Password, request.Language, false)
}

func (service *serviceImpl) createUser(c *echo.Context, name, email, password, language string, isConfirmed bool) error {
	count, err := service.userCollection.Find(bson.M{"email": email}).Count()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, util.CreateErrResponse(err))
	}

	if count > 0 {
		return c.JSON(http.StatusConflict, util.CreateErrResponse(errors.New("This email address is already being used.")))
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, util.CreateErrResponse(err))
	}

	userDoc := collections.UserDocument{
		Id:          bson.NewObjectId(),
		Name:        name,
		Email:       email,
		HashedPass:  hashedPass,
		Language:    language,
		CreatedAt:   time.Now(),
		IsSuspended: false,
	}

	var tokenStr string

	if isConfirmed {
		userDoc.ConfirmedAt = time.Now()

	} else {
		userDoc.ConfirmationKey = uuid.NewV4().String()

		token := tokens.ConfirmationToken{Key: userDoc.ConfirmationKey}

		tokenStr, err = token.ToString(service.jwtKey)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, util.CreateErrResponse(err))
		}
	}

	err = service.userCollection.Insert(userDoc)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, util.CreateErrResponse(err))
	}

	return c.JSON(http.StatusOK, util.CreateOkResponse(bson.M{"ConfirmationToken": tokenStr}))
}