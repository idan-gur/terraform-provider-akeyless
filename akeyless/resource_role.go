package akeyless

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/akeylesslabs/akeyless-go/v2"
	"github.com/akeylesslabs/terraform-provider-akeyless/akeyless/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

func resourceRole() *schema.Resource {
	return &schema.Resource{
		Description: "Role Resource",
		Create:      resourceRoleCreate,
		Read:        resourceRoleRead,
		Update:      resourceRoleUpdate,
		Delete:      resourceRoleDelete,
		Importer: &schema.ResourceImporter{
			State: resourceRoleImport,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Role name",
			},
			"comment": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Comment about the role",
				Default:     "",
			},
			"assoc_auth_method": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Create an association between role and auth method",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"am_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The auth method to associate",
						},
						"sub_claims": {
							Type:        schema.TypeMap,
							Optional:    true,
							Description: "key/val of sub claims, e.g group=admins,developers",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"rules": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Set a rule to a role",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"capability": {
							Type:        schema.TypeSet,
							Required:    true,
							Description: "List of the approved/denied capabilities in the path options: [read, create, update, delete, list, deny]",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"path": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The path the rule refers to",
						},
						"rule_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "item-rule, role-rule or auth-method-rule",
							Default:     "item-rule",
						},
					},
				},
			},
			"assoc_auth_method_with_rules": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceRoleCreate(d *schema.ResourceData, m interface{}) error {
	provider := m.(providerMeta)
	client := *provider.client
	token := *provider.token

	name := d.Get("name").(string)
	comment := d.Get("comment").(string)

	var apiErr akeyless.GenericOpenAPIError
	ctx := context.Background()
	body := akeyless.CreateRole{
		Name:    name,
		Comment: akeyless.PtrString(comment),
		Token:   &token,
	}

	_, _, err := client.CreateRole(ctx).Body(body).Execute()
	if err != nil {
		if errors.As(err, &apiErr) {
			return fmt.Errorf("can't create Role: %v", string(apiErr.Body()))
		}
		return fmt.Errorf("can't create Role: %v", err)
	}

	assocAuthMethod := d.Get("assoc_auth_method").([]interface{})
	if assocAuthMethod != nil {
		for _, v := range assocAuthMethod {
			assoc := v.(map[string]interface{})
			authMethodName := assoc["am_name"].(string)
			subClaims := assoc["sub_claims"].(map[string]interface{})
			sc := make(map[string]string, len(subClaims))
			for k, v := range subClaims {
				sc[k] = v.(string)
			}

			asBody := akeyless.AssocRoleAuthMethod{
				RoleName:  name,
				AmName:    authMethodName,
				SubClaims: &sc,
				Token:     &token,
			}

			_, _, err = client.AssocRoleAuthMethod(ctx).Body(asBody).Execute()
			if err != nil {
				if errors.As(err, &apiErr) {
					return fmt.Errorf("can't create association: %v", string(apiErr.Body()))
				}
				return fmt.Errorf("can't create association: %v", err)
			}
		}
	}

	roleRules := d.Get("rules").([]interface{})
	if roleRules != nil {
		for _, v := range roleRules {
			roles := v.(map[string]interface{})
			capability := roles["capability"].(*schema.Set)
			path := roles["path"].(string)
			ruleType := roles["rule_type"].(string)

			setRoleRule := akeyless.SetRoleRule{
				RoleName:   name,
				Capability: common.ExpandStringList(capability.List()),
				Path:       path,
				RuleType:   akeyless.PtrString(ruleType),
				Token:      &token,
			}
			_, _, err := client.SetRoleRule(ctx).Body(setRoleRule).Execute()
			if err != nil {
				if errors.As(err, &apiErr) {
					return fmt.Errorf("can't set rules: %v", string(apiErr.Body()))
				}
				return fmt.Errorf("can't set rules: %v", err)
			}
		}
	}

	d.SetId(name)

	return nil
}

func resourceRoleRead(d *schema.ResourceData, m interface{}) error {
	provider := m.(providerMeta)
	client := *provider.client
	token := *provider.token

	name := d.Id()
	body := akeyless.GetRole{
		Name:  name,
		Token: &token,
	}

	role, err := getRole(d, client, body)
	if err != nil {
		return err
	}

	roleAsJson, err := json.Marshal(role)
	if err != nil {
		return err
	}

	err = d.Set("assoc_auth_method_with_rules", string(roleAsJson))
	if err != nil {
		return err
	}

	return nil
}

func resourceRoleUpdate(d *schema.ResourceData, m interface{}) error {
	provider := m.(providerMeta)
	client := *provider.client
	token := *provider.token

	name := d.Get("name").(string)
	var apiErr akeyless.GenericOpenAPIError
	ctx := context.Background()

	body := akeyless.GetRole{
		Name:  name,
		Token: &token,
	}
	role, err := getRole(d, client, body)
	if err != nil {
		return err
	}

	for _, v := range *role.RoleAuthMethodsAssoc {
		association := akeyless.DeleteRoleAssociation{
			AssocId: *v.AssocId,
			Token:   &token,
		}

		_, res, err := client.DeleteRoleAssociation(ctx).Body(association).Execute()
		if err != nil {
			if errors.As(err, &apiErr) {
				if res.StatusCode != http.StatusNotFound {
					return fmt.Errorf("can't delete Role association: %v", string(apiErr.Body()))
				}
			} else {
				return fmt.Errorf("can't delete Role association: %v", err)
			}
		}
	}

	for _, v := range *role.Rules.PathRules {
		rule := akeyless.DeleteRoleRule{
			RoleName: name,
			Path:     *v.Path,
			RuleType: v.Type,
			Token:    &token,
		}

		_, res, err := client.DeleteRoleRule(ctx).Body(rule).Execute()
		if err != nil {
			if errors.As(err, &apiErr) {
				if res.StatusCode != http.StatusNotFound {
					return fmt.Errorf("can't delete rule: %v", string(apiErr.Body()))
				}
			} else {
				return fmt.Errorf("can't delete rule: %v", err)
			}
		}
	}

	assocAuthMethod := d.Get("assoc_auth_method").([]interface{})
	if assocAuthMethod != nil {
		for _, v := range assocAuthMethod {
			assoc := v.(map[string]interface{})
			authMethodName := assoc["am_name"].(string)
			subClaims := assoc["sub_claims"].(map[string]interface{})
			sc := make(map[string]string, len(subClaims))
			for k, v := range subClaims {
				sc[k] = v.(string)
			}

			asBody := akeyless.AssocRoleAuthMethod{
				RoleName:  name,
				AmName:    authMethodName,
				SubClaims: &sc,
				Token:     &token,
			}

			_, res, err := client.AssocRoleAuthMethod(ctx).Body(asBody).Execute()
			if err != nil {
				if errors.As(err, &apiErr) {
					if res.StatusCode != http.StatusConflict {
						return fmt.Errorf("can't create association: %v", string(apiErr.Body()))
					}
				} else {
					return fmt.Errorf("can't create association: %v", err)
				}
			}
		}
	}

	roleRules := d.Get("rules").([]interface{})
	if roleRules != nil {
		for _, v := range roleRules {
			roles := v.(map[string]interface{})
			capability := roles["capability"].(*schema.Set)
			path := roles["path"].(string)
			ruleType := roles["rule_type"].(string)

			setRoleRule := akeyless.SetRoleRule{
				RoleName:   name,
				Capability: common.ExpandStringList(capability.List()),
				Path:       path,
				RuleType:   akeyless.PtrString(ruleType),
				Token:      &token,
			}
			_, res, err := client.SetRoleRule(ctx).Body(setRoleRule).Execute()
			if err != nil {
				if errors.As(err, &apiErr) {
					if res.StatusCode != http.StatusConflict {
						return fmt.Errorf("can't set rules: %v", string(apiErr.Body()))
					}

				}
				return fmt.Errorf("can't set rules: %v", err)
			}
		}
	}

	d.SetId(name)

	return nil
}

func resourceRoleDelete(d *schema.ResourceData, m interface{}) error {
	provider := m.(providerMeta)
	client := *provider.client
	token := *provider.token

	name := d.Id()

	body := akeyless.GetRole{
		Name:  name,
		Token: &token,
	}

	ctx := context.Background()
	role, err := getRole(d, client, body)
	if err != nil {
		return err
	}

	for _, v := range role.GetRoleAuthMethodsAssoc() {
		deleteRoleAssociation := akeyless.DeleteRoleAssociation{
			AssocId: *v.AssocId,
			Token:   &token,
		}
		_, res, err := client.DeleteRoleAssociation(ctx).Body(deleteRoleAssociation).Execute()
		if err != nil {
			var apiErr akeyless.GenericOpenAPIError
			if errors.As(err, &apiErr) {
				if res.StatusCode != http.StatusNotFound {
					return err
				}
			} else {
				return err
			}
		}
	}

	rules := role.GetRules()
	for _, v := range rules.GetPathRules() {
		deleteRoleRule := akeyless.DeleteRoleRule{
			RoleName: name,
			Path:     *v.Path,
			RuleType: v.Type,
			Token:    &token,
		}

		_, res, err := client.DeleteRoleRule(ctx).Body(deleteRoleRule).Execute()
		if err != nil {
			var apiErr akeyless.GenericOpenAPIError
			if errors.As(err, &apiErr) {
				if res.StatusCode != http.StatusNotFound {
					return err
				}
			} else {
				return err
			}
		}
	}

	deleteRole := akeyless.DeleteRole{
		Name:  name,
		Token: &token,
	}

	_, _, err = client.DeleteRole(ctx).Body(deleteRole).Execute()
	if err != nil {
		return err
	}

	return nil
}

func resourceRoleImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	provider := m.(providerMeta)
	client := *provider.client
	token := *provider.token

	name := d.Id()

	item := akeyless.GetRole{
		Name:  name,
		Token: &token,
	}

	ctx := context.Background()
	_, _, err := client.GetRole(ctx).Body(item).Execute()
	if err != nil {
		return nil, err
	}

	err = d.Set("name", name)
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func getRole(d *schema.ResourceData, client akeyless.V2ApiService, body akeyless.GetRole) (akeyless.Role, error) {
	var apiErr akeyless.GenericOpenAPIError
	ctx := context.Background()
	role, res, err := client.GetRole(ctx).Body(body).Execute()
	if err != nil {
		if errors.As(err, &apiErr) {
			if res.StatusCode == http.StatusNotFound {
				// The secret was deleted outside of the current Terraform workspace, so invalidate this resource
				d.SetId("")
				return akeyless.Role{}, nil
			}
			return akeyless.Role{}, fmt.Errorf("can't get Role value: %v", string(apiErr.Body()))
		}
		return akeyless.Role{}, fmt.Errorf("can't get Role value: %v", err)
	}
	return role, nil
}