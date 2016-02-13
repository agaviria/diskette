package user

import (
	"diskette/collections"
	"diskette/tokens"
	"diskette/util"
	"errors"
	"net/http"
	"time"

	"diskette/vendor/github.com/satori/go.uuid"

	"github.com/labstack/echo"
	"golang.org/x/crypto/bcrypt"
	"labix.org/v2/mgo/bson"
)

// http POST localhost:5025/user/signup email=joe.doe@gmail.com password=abc profile:='{"name": "Joe Doe", "language": "en" }'
func (service *serviceImpl) Signup(c *echo.Context) error {
	var request struct {
		Email    string                 `json:"email"`
		Password string                 `json:"password"`
		Profile  map[string]interface{} `json:"profile"`
	}
	c.Bind(&request)

	if request.Email == "" {
		return c.JSON(http.StatusBadRequest, util.CreateErrResponse(errors.New("Missing parameter 'email'")))
	}

	if request.Password == "" {
		return c.JSON(http.StatusBadRequest, util.CreateErrResponse(errors.New("Missing parameter 'password'")))
	}

	if request.Profile == nil {
		return c.JSON(http.StatusBadRequest, util.CreateErrResponse(errors.New("Missing parameter 'profile'")))
	}

	return service.createUser(c, request.Email, request.Password, request.Profile, false)
}

func (service *serviceImpl) createUser(c *echo.Context, email, password string, profile map[string]interface{}, isConfirmed bool) error {
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
		Email:       email,
		HashedPass:  hashedPass,
		Profile:     profile,
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
