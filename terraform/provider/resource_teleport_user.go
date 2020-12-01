package provider

import (
	"context"
	"strings"

	"github.com/gravitational/teleport/lib/auth"
	"github.com/gravitational/teleport/lib/services"

	"github.com/gravitational/trace"

	"github.com/hashicorp/terraform/helper/schema"
)

// teleport_user resource definition
func resourceTeleportUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceTeleportUserUpsert,
		Read:   resourceTeleportUserRead,
		Update: resourceTeleportUserUpsert,
		Delete: resourceTeleportUserDelete,
		Exists: resourceTeleportUserExists,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"roles": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"traits": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
	}
}

func resourceTeleportUserUpsert(d *schema.ResourceData, m interface{}) error {
	client := m.(*auth.Client)

	name := d.Get("name").(string)

	tfRoles := d.Get("roles").([]interface{})
	roles := make([]string, len(tfRoles))

	for i, tfRole := range tfRoles {
		roles[i] = tfRole.(string)
	}

	tfTraits := d.Get("traits").(map[string]interface{})
	traits := make(map[string][]string)

	for k, tfTraitString := range tfTraits {
		traits[k] = strings.Split(tfTraitString.(string), " ")
	}

	user, err := services.NewUser(name)
	if err != nil {
		return trace.Wrap(err)
	}

	user.SetRoles(roles)
	user.SetTraits(traits)

	err = client.UpsertUser(user)
	if err != nil {
		return trace.Wrap(err)
	}

	d.SetId(name)
	return nil
}

func resourceTeleportUserRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*auth.Client)
	name := d.Get("name").(string)

	u, err := client.GetUser(name, false)
	if err != nil {
		return trace.Wrap(err)
	}

	user := u.(services.User)

	traits := user.GetTraits()
	tfTraits := map[string]string{}

	for k, trait := range traits {
		tfTraits[k] = strings.Join(trait, " ")
	}

	d.Set("roles", user.GetRoles())
	d.Set("traits", tfTraits)

	return nil
}

func resourceTeleportUserDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*auth.Client)
	name := d.Get("name").(string)

	err := client.DeleteUser(context.TODO(), name)
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}

func resourceTeleportUserExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := m.(*auth.Client)
	name := d.Get("name").(string)

	user, err := client.GetUser(name, false)

	if err != nil {
		return false, err
	}

	return user != nil, nil
}